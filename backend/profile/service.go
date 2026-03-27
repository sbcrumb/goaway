package profile

import (
	"context"
	"goaway/backend/database"
	"goaway/backend/logging"
	model "goaway/backend/dns/server/models"
	"net"
	"sort"
	"strings"
	"sync"
)

var log = logging.GetLogger()

type Service struct {
	repo Repository

	defaultProfileID uint

	// profileID -> set of blocked domains (active sources + custom blacklist)
	blockCaches map[uint]map[string]bool
	blockMu     sync.RWMutex

	// profileID -> set of whitelisted domains
	whitelistCaches map[uint]map[string]bool
	wlMu            sync.RWMutex

	// Subnet rules sorted most-specific first (largest prefix length)
	subnetRules []subnetRule
	subnetMu    sync.RWMutex
}

type subnetRule struct {
	net       *net.IPNet
	profileID uint
	ones      int // prefix length, for sorting
}

func NewService(repo Repository) *Service {
	return &Service{
		repo:            repo,
		blockCaches:     make(map[uint]map[string]bool),
		whitelistCaches: make(map[uint]map[string]bool),
	}
}

// Initialize seeds the Default profile if missing, syncs sources, and builds all caches.
func (s *Service) Initialize(ctx context.Context) error {
	defaultProfile, err := s.repo.GetDefaultProfile(ctx)
	if err != nil {
		// Default profile doesn't exist yet — create it
		p := &database.Profile{Name: "Default", IsDefault: true}
		if err := s.repo.CreateProfile(ctx, p); err != nil {
			return err
		}
		defaultProfile = p
		log.Info("Created Default profile (id=%d)", defaultProfile.ID)
	}
	s.defaultProfileID = defaultProfile.ID

	// Ensure every existing source has a profile_sources row for every profile
	sources, err := s.repo.GetAllSources(ctx)
	if err != nil {
		return err
	}
	profiles, err := s.repo.GetAllProfiles(ctx)
	if err != nil {
		return err
	}
	for _, p := range profiles {
		if err := s.repo.SyncProfileSources(ctx, p.ID, sources); err != nil {
			return err
		}
	}

	if err := s.RebuildAllCaches(ctx); err != nil {
		return err
	}
	return s.RebuildSubnetRules(ctx)
}

func (s *Service) DefaultProfileID() uint {
	return s.defaultProfileID
}

// ResolveProfileForClient returns the effective profile ID for a client.
// Priority: individual IP assignment > subnet match > Default.
func (s *Service) ResolveProfileForClient(client *model.Client) uint {
	if client.ProfileID != nil {
		return *client.ProfileID
	}

	ip := net.ParseIP(client.IP)
	if ip != nil {
		s.subnetMu.RLock()
		defer s.subnetMu.RUnlock()
		for _, rule := range s.subnetRules {
			if rule.net.Contains(ip) {
				return rule.profileID
			}
		}
	}

	return s.defaultProfileID
}

// IsBlockedForProfile checks if a domain is blocked under a given profile.
func (s *Service) IsBlockedForProfile(profileID uint, domain string) bool {
	domain = strings.TrimSuffix(domain, ".")
	s.blockMu.RLock()
	defer s.blockMu.RUnlock()
	cache, ok := s.blockCaches[profileID]
	if !ok {
		return false
	}
	return cache[domain]
}

// IsWhitelistedForProfile checks if a domain is whitelisted under a given profile.
func (s *Service) IsWhitelistedForProfile(profileID uint, domain string) bool {
	domain = strings.TrimSuffix(domain, ".")
	s.wlMu.RLock()
	defer s.wlMu.RUnlock()
	cache, ok := s.whitelistCaches[profileID]
	if !ok {
		return false
	}
	return cache[domain]
}

// RebuildAllCaches rebuilds block and whitelist caches for every profile.
func (s *Service) RebuildAllCaches(ctx context.Context) error {
	profiles, err := s.repo.GetAllProfiles(ctx)
	if err != nil {
		return err
	}
	for _, p := range profiles {
		if err := s.RebuildCacheForProfile(ctx, p.ID); err != nil {
			log.Warning("Failed to rebuild cache for profile %d (%s): %v", p.ID, p.Name, err)
		}
	}
	return nil
}

// RebuildCacheForProfile rebuilds block and whitelist caches for a single profile.
func (s *Service) RebuildCacheForProfile(ctx context.Context, profileID uint) error {
	// Build block cache: active source domains + custom blacklist
	sourceDomains, err := s.repo.GetActiveSourceDomains(ctx, profileID)
	if err != nil {
		return err
	}
	customDomains, _, err := s.repo.GetProfileCustomBlacklist(ctx, profileID, 1, 1000000, "")
	if err != nil {
		return err
	}

	blockCache := make(map[string]bool, len(sourceDomains)+len(customDomains))
	for _, d := range sourceDomains {
		blockCache[strings.TrimSuffix(d, ".")] = true
	}
	for _, d := range customDomains {
		blockCache[strings.TrimSuffix(d, ".")] = true
	}

	s.blockMu.Lock()
	s.blockCaches[profileID] = blockCache
	s.blockMu.Unlock()

	// Build whitelist cache
	wlDomains, err := s.repo.GetProfileWhitelist(ctx, profileID)
	if err != nil {
		return err
	}
	wlCache := make(map[string]bool, len(wlDomains))
	for _, d := range wlDomains {
		wlCache[strings.TrimSuffix(d, ".")] = true
	}

	s.wlMu.Lock()
	s.whitelistCaches[profileID] = wlCache
	s.wlMu.Unlock()

	log.Debug("Rebuilt cache for profile %d: %d blocked, %d whitelisted", profileID, len(blockCache), len(wlCache))
	return nil
}

// RebuildSubnetRules reloads subnet assignments from DB and re-parses them.
func (s *Service) RebuildSubnetRules(ctx context.Context) error {
	subnets, err := s.repo.GetAllSubnets(ctx)
	if err != nil {
		return err
	}

	rules := make([]subnetRule, 0, len(subnets))
	for _, sub := range subnets {
		_, ipNet, err := net.ParseCIDR(sub.CIDR)
		if err != nil {
			log.Warning("Invalid CIDR in subnet_profiles id=%d: %s — skipping", sub.ID, sub.CIDR)
			continue
		}
		ones, _ := ipNet.Mask.Size()
		rules = append(rules, subnetRule{net: ipNet, profileID: sub.ProfileID, ones: ones})
	}

	// Sort most-specific first so /24 is checked before /16
	sort.Slice(rules, func(i, j int) bool {
		return rules[i].ones > rules[j].ones
	})

	s.subnetMu.Lock()
	s.subnetRules = rules
	s.subnetMu.Unlock()

	log.Debug("Loaded %d subnet profile rules", len(rules))
	return nil
}

// --- Profile management helpers called by API handlers ---

func (s *Service) ListProfiles(ctx context.Context) ([]database.Profile, error) {
	return s.repo.GetAllProfiles(ctx)
}

func (s *Service) GetAllSources(ctx context.Context) ([]database.Source, error) {
	return s.repo.GetAllSources(ctx)
}

func (s *Service) GetProfile(ctx context.Context, id uint) (*ProfileDetail, error) {
	p, err := s.repo.GetProfileByID(ctx, id)
	if err != nil {
		return nil, err
	}
	profileSources, err := s.repo.GetProfileSources(ctx, id)
	if err != nil {
		return nil, err
	}

	// Build a map of sourceID -> Source so we can populate Name and URL
	allSources, err := s.repo.GetAllSources(ctx)
	if err != nil {
		return nil, err
	}
	sourceMap := make(map[uint]database.Source, len(allSources))
	for _, src := range allSources {
		sourceMap[src.ID] = src
	}

	detail := &ProfileDetail{
		ID:        p.ID,
		Name:      p.Name,
		IsDefault: p.IsDefault,
		Sources:   make([]ProfileSourceStatus, len(profileSources)),
	}
	for i, ps := range profileSources {
		src := sourceMap[ps.SourceID]
		detail.Sources[i] = ProfileSourceStatus{
			SourceID: ps.SourceID,
			Name:     src.Name,
			URL:      src.URL,
			Active:   ps.Active,
		}
	}
	return detail, nil
}

func (s *Service) CreateProfile(ctx context.Context, name string, allSources []database.Source) (*database.Profile, error) {
	p := &database.Profile{Name: name, IsDefault: false}
	if err := s.repo.CreateProfile(ctx, p); err != nil {
		return nil, err
	}
	// Seed profile_sources rows for all existing sources
	if err := s.repo.SyncProfileSources(ctx, p.ID, allSources); err != nil {
		return nil, err
	}
	if err := s.RebuildCacheForProfile(ctx, p.ID); err != nil {
		log.Warning("Failed to build initial cache for new profile %d: %v", p.ID, err)
	}
	return p, nil
}

func (s *Service) RenameProfile(ctx context.Context, id uint, name string) error {
	return s.repo.UpdateProfileName(ctx, id, name)
}

func (s *Service) DeleteProfile(ctx context.Context, id uint) error {
	if err := s.repo.DeleteProfile(ctx, id); err != nil {
		return err
	}
	s.blockMu.Lock()
	delete(s.blockCaches, id)
	s.blockMu.Unlock()
	s.wlMu.Lock()
	delete(s.whitelistCaches, id)
	s.wlMu.Unlock()
	return nil
}

func (s *Service) ToggleProfileSource(ctx context.Context, profileID, sourceID uint, active bool) error {
	ps := &database.ProfileSource{ProfileID: profileID, SourceID: sourceID, Active: active}
	if err := s.repo.UpsertProfileSource(ctx, ps); err != nil {
		return err
	}
	return s.RebuildCacheForProfile(ctx, profileID)
}

func (s *Service) GetProfileCustomBlacklist(ctx context.Context, profileID uint, page, pageSize int, search string) ([]string, int64, error) {
	return s.repo.GetProfileCustomBlacklist(ctx, profileID, page, pageSize, search)
}

func (s *Service) AddProfileCustomBlacklist(ctx context.Context, profileID uint, domains []string) error {
	if err := s.repo.AddProfileCustomBlacklist(ctx, profileID, domains); err != nil {
		return err
	}
	return s.RebuildCacheForProfile(ctx, profileID)
}

func (s *Service) RemoveProfileCustomBlacklist(ctx context.Context, profileID uint, domain string) error {
	if err := s.repo.RemoveProfileCustomBlacklist(ctx, profileID, domain); err != nil {
		return err
	}
	return s.RebuildCacheForProfile(ctx, profileID)
}

func (s *Service) GetProfileWhitelist(ctx context.Context, profileID uint) ([]string, error) {
	return s.repo.GetProfileWhitelist(ctx, profileID)
}

func (s *Service) AddProfileWhitelist(ctx context.Context, profileID uint, domain string) error {
	if err := s.repo.AddProfileWhitelist(ctx, profileID, domain); err != nil {
		return err
	}
	return s.RebuildCacheForProfile(ctx, profileID)
}

func (s *Service) RemoveProfileWhitelist(ctx context.Context, profileID uint, domain string) error {
	if err := s.repo.RemoveProfileWhitelist(ctx, profileID, domain); err != nil {
		return err
	}
	return s.RebuildCacheForProfile(ctx, profileID)
}

func (s *Service) ListSubnets(ctx context.Context) ([]database.SubnetProfile, error) {
	return s.repo.GetAllSubnets(ctx)
}

func (s *Service) CreateSubnet(ctx context.Context, cidr string, profileID uint) (*database.SubnetProfile, error) {
	if _, _, err := net.ParseCIDR(cidr); err != nil {
		return nil, err
	}
	sub := &database.SubnetProfile{CIDR: cidr, ProfileID: profileID}
	if err := s.repo.CreateSubnet(ctx, sub); err != nil {
		return nil, err
	}
	if err := s.RebuildSubnetRules(ctx); err != nil {
		log.Warning("Failed to rebuild subnet rules: %v", err)
	}
	return sub, nil
}

func (s *Service) UpdateSubnet(ctx context.Context, id uint, cidr string, profileID uint) error {
	if _, _, err := net.ParseCIDR(cidr); err != nil {
		return err
	}
	if err := s.repo.UpdateSubnet(ctx, id, cidr, profileID); err != nil {
		return err
	}
	return s.RebuildSubnetRules(ctx)
}

func (s *Service) DeleteSubnet(ctx context.Context, id uint) error {
	if err := s.repo.DeleteSubnet(ctx, id); err != nil {
		return err
	}
	return s.RebuildSubnetRules(ctx)
}

func (s *Service) SetClientProfile(ctx context.Context, ip string, profileID *uint) error {
	return s.repo.SetClientProfile(ctx, ip, profileID)
}
