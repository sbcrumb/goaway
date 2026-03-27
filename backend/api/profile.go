package api

import (
	"context"
	"goaway/backend/profile"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

func (api *API) registerProfileRoutes() {
	api.routes.GET("/profiles", api.listProfiles)
	api.routes.POST("/profiles", api.createProfile)
	api.routes.GET("/profiles/:id", api.getProfile)
	api.routes.PUT("/profiles/:id/name", api.renameProfile)
	api.routes.DELETE("/profiles/:id", api.deleteProfile)

	api.routes.GET("/profiles/:id/sources", api.getProfileSources)
	api.routes.PUT("/profiles/:id/sources/:sourceId/toggle", api.toggleProfileSource)

	api.routes.GET("/profiles/:id/blacklist", api.getProfileBlacklist)
	api.routes.POST("/profiles/:id/blacklist", api.addProfileBlacklist)
	api.routes.DELETE("/profiles/:id/blacklist", api.removeProfileBlacklist)

	api.routes.GET("/profiles/:id/whitelist", api.getProfileWhitelist)
	api.routes.POST("/profiles/:id/whitelist", api.addProfileWhitelist)
	api.routes.DELETE("/profiles/:id/whitelist", api.removeProfileWhitelist)

	api.routes.GET("/subnets", api.listSubnets)
	api.routes.POST("/subnets", api.createSubnet)
	api.routes.PUT("/subnets/:id", api.updateSubnet)
	api.routes.DELETE("/subnets/:id", api.deleteSubnet)

	api.routes.PUT("/client/:ip/profile/:profileId", api.assignClientProfile)
	api.routes.DELETE("/client/:ip/profile", api.removeClientProfile)
}

func (api *API) profileService() *profile.Service {
	return api.DNS.ProfileService
}

func parseProfileID(c *gin.Context) (uint, bool) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid profile id"})
		return 0, false
	}
	return uint(id), true
}

// --- Profile CRUD ---

func (api *API) listProfiles(c *gin.Context) {
	profiles, err := api.profileService().ListProfiles(context.Background())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, profiles)
}

func (api *API) createProfile(c *gin.Context) {
	var body struct {
		Name string `json:"name" binding:"required"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	allSources, err := api.profileService().GetAllSources(context.Background())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	p, err := api.profileService().CreateProfile(context.Background(), body.Name, allSources)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, p)
}

func (api *API) getProfile(c *gin.Context) {
	id, ok := parseProfileID(c)
	if !ok {
		return
	}
	detail, err := api.profileService().GetProfile(context.Background(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, detail)
}

func (api *API) renameProfile(c *gin.Context) {
	id, ok := parseProfileID(c)
	if !ok {
		return
	}
	var body struct {
		Name string `json:"name" binding:"required"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := api.profileService().RenameProfile(context.Background(), id, body.Name); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "profile renamed"})
}

func (api *API) deleteProfile(c *gin.Context) {
	id, ok := parseProfileID(c)
	if !ok {
		return
	}
	if err := api.profileService().DeleteProfile(context.Background(), id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "profile deleted"})
}

// --- Source management ---

func (api *API) getProfileSources(c *gin.Context) {
	id, ok := parseProfileID(c)
	if !ok {
		return
	}
	detail, err := api.profileService().GetProfile(context.Background(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, detail.Sources)
}

func (api *API) toggleProfileSource(c *gin.Context) {
	id, ok := parseProfileID(c)
	if !ok {
		return
	}
	sourceID, err := strconv.ParseUint(c.Param("sourceId"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid source id"})
		return
	}
	var body struct {
		Active bool `json:"active"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := api.profileService().ToggleProfileSource(context.Background(), id, uint(sourceID), body.Active); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "source toggled"})
}

// --- Custom blacklist ---

func (api *API) getProfileBlacklist(c *gin.Context) {
	id, ok := parseProfileID(c)
	if !ok {
		return
	}
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "50"))
	search := c.Query("search")

	domains, total, err := api.profileService().GetProfileCustomBlacklist(context.Background(), id, page, pageSize, search)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"domains": domains, "total": total})
}

func (api *API) addProfileBlacklist(c *gin.Context) {
	id, ok := parseProfileID(c)
	if !ok {
		return
	}
	var body struct {
		Domains []string `json:"domains" binding:"required"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := api.profileService().AddProfileCustomBlacklist(context.Background(), id, body.Domains); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"message": "domains added"})
}

func (api *API) removeProfileBlacklist(c *gin.Context) {
	id, ok := parseProfileID(c)
	if !ok {
		return
	}
	domain := c.Query("domain")
	if domain == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "domain query param required"})
		return
	}
	if err := api.profileService().RemoveProfileCustomBlacklist(context.Background(), id, domain); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "domain removed"})
}

// --- Whitelist ---

func (api *API) getProfileWhitelist(c *gin.Context) {
	id, ok := parseProfileID(c)
	if !ok {
		return
	}
	domains, err := api.profileService().GetProfileWhitelist(context.Background(), id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"domains": domains})
}

func (api *API) addProfileWhitelist(c *gin.Context) {
	id, ok := parseProfileID(c)
	if !ok {
		return
	}
	var body struct {
		Domain string `json:"domain" binding:"required"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := api.profileService().AddProfileWhitelist(context.Background(), id, body.Domain); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"message": "domain whitelisted"})
}

func (api *API) removeProfileWhitelist(c *gin.Context) {
	id, ok := parseProfileID(c)
	if !ok {
		return
	}
	domain := c.Query("domain")
	if domain == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "domain query param required"})
		return
	}
	if err := api.profileService().RemoveProfileWhitelist(context.Background(), id, domain); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "domain removed from whitelist"})
}

// --- Subnets ---

func (api *API) listSubnets(c *gin.Context) {
	subnets, err := api.profileService().ListSubnets(context.Background())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, subnets)
}

func (api *API) createSubnet(c *gin.Context) {
	var body struct {
		CIDR      string `json:"cidr" binding:"required"`
		ProfileID uint   `json:"profileId" binding:"required"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	sub, err := api.profileService().CreateSubnet(context.Background(), body.CIDR, body.ProfileID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, sub)
}

func (api *API) updateSubnet(c *gin.Context) {
	subnetID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid subnet id"})
		return
	}
	var body struct {
		CIDR      string `json:"cidr" binding:"required"`
		ProfileID uint   `json:"profileId" binding:"required"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := api.profileService().UpdateSubnet(context.Background(), uint(subnetID), body.CIDR, body.ProfileID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "subnet updated"})
}

func (api *API) deleteSubnet(c *gin.Context) {
	subnetID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid subnet id"})
		return
	}
	if err := api.profileService().DeleteSubnet(context.Background(), uint(subnetID)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "subnet deleted"})
}

// --- Client profile assignment ---

func (api *API) assignClientProfile(c *gin.Context) {
	ip := c.Param("ip")
	profileID, err := strconv.ParseUint(c.Param("profileId"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid profile id"})
		return
	}
	pid := uint(profileID)
	if err := api.profileService().SetClientProfile(context.Background(), ip, &pid); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	_ = api.DNS.PopulateClientCaches()
	c.JSON(http.StatusOK, gin.H{"message": "profile assigned"})
}

func (api *API) removeClientProfile(c *gin.Context) {
	ip := c.Param("ip")
	if err := api.profileService().SetClientProfile(context.Background(), ip, nil); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	_ = api.DNS.PopulateClientCaches()
	c.JSON(http.StatusOK, gin.H{"message": "profile removed"})
}
