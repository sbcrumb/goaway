import { ProfileDetail, type ProfileDetail as ProfileDetailType } from "@/app/profiles/ProfileDetail";
import { Button } from "@/components/ui/button";
import { Card } from "@/components/ui/card";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogTrigger
} from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { Separator } from "@/components/ui/separator";
import { DeleteRequest, GetRequest, PostRequest, PutRequest } from "@/util";
import {
  NetworkIcon,
  PlusIcon,
  ShieldIcon,
  TrashIcon
} from "@phosphor-icons/react";
import { useEffect, useState } from "react";
import { toast } from "sonner";

type SubnetRule = {
  id: number;
  cidr: string;
  profileId: number;
  profileName: string;
};

export function Profiles() {
  const [profiles, setProfiles] = useState<ProfileDetailType[]>([]);
  const [subnets, setSubnets] = useState<SubnetRule[]>([]);
  const [selectedProfile, setSelectedProfile] =
    useState<ProfileDetailType | null>(null);
  const [sheetOpen, setSheetOpen] = useState(false);

  const [createOpen, setCreateOpen] = useState(false);
  const [newProfileName, setNewProfileName] = useState("");

  const [subnetCIDR, setSubnetCIDR] = useState("");
  const [subnetProfileId, setSubnetProfileId] = useState<number | "">("");

  useEffect(() => {
    fetchProfiles();
    fetchSubnets();
  }, []);

  const fetchProfiles = async () => {
    const [code, data] = await GetRequest("profiles");
    if (code === 200 && Array.isArray(data)) {
      setProfiles(data);
    }
  };

  const fetchSubnets = async () => {
    const [code, data] = await GetRequest("subnets");
    if (code === 200 && Array.isArray(data)) {
      setSubnets(data);
    }
  };

  const handleCreateProfile = async () => {
    const name = newProfileName.trim();
    if (!name) return;
    const [code, data] = await PostRequest("profiles", { name });
    if (code === 201) {
      setProfiles((prev) => [...prev, data]);
      setNewProfileName("");
      setCreateOpen(false);
      toast.success(`Profile "${name}" created`);
    }
  };

  const handleDeleteProfile = async (profile: ProfileDetailType) => {
    if (profile.isDefault) {
      toast.warning("Cannot delete the Default profile");
      return;
    }
    const [code] = await DeleteRequest(`profiles/${profile.id}`, null);
    if (code === 200) {
      setProfiles((prev) => prev.filter((p) => p.id !== profile.id));
      toast.success(`Profile "${profile.name}" deleted`);
    }
  };

  const handleRenamed = (id: number, name: string) => {
    setProfiles((prev) =>
      prev.map((p) => (p.id === id ? { ...p, name } : p))
    );
    if (selectedProfile?.id === id) {
      setSelectedProfile((prev) => (prev ? { ...prev, name } : prev));
    }
  };

  const handleOpenProfile = (profile: ProfileDetailType) => {
    setSelectedProfile(profile);
    setSheetOpen(true);
  };

  const handleAddSubnet = async () => {
    const cidr = subnetCIDR.trim();
    if (!cidr || subnetProfileId === "") return;
    const [code, data] = await PostRequest("subnets", {
      cidr,
      profileId: subnetProfileId
    });
    if (code === 201) {
      setSubnets((prev) => [...prev, data]);
      setSubnetCIDR("");
      setSubnetProfileId("");
      toast.success(`Subnet ${cidr} assigned`);
    }
  };

  const handleDeleteSubnet = async (id: number) => {
    const [code] = await DeleteRequest(`subnets/${id}`, null);
    if (code === 200) {
      setSubnets((prev) => prev.filter((s) => s.id !== id));
    }
  };

  const activeSourcesCount = (profile: ProfileDetailType) =>
    (profile.sources ?? []).filter((s) => s.active).length;

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between flex-wrap gap-3">
        <div className="flex gap-3">
          <div className="flex items-center gap-2 px-4 py-1 bg-accent border-b rounded-t-sm border-b-blue-400">
            <div className="w-2 h-2 bg-blue-500 rounded-full" />
            <span className="text-muted-foreground text-sm">Profiles:</span>
            <span className="font-semibold">{profiles.length}</span>
          </div>
          <div className="flex items-center gap-2 px-4 py-1 bg-accent border-b rounded-t-sm border-b-purple-400">
            <div className="w-2 h-2 bg-purple-500 rounded-full" />
            <span className="text-muted-foreground text-sm">Subnets:</span>
            <span className="font-semibold">{subnets.length}</span>
          </div>
        </div>

        <Dialog open={createOpen} onOpenChange={setCreateOpen}>
          <DialogTrigger asChild>
            <Button>
              <PlusIcon className="mr-2" size={16} />
              New Profile
            </Button>
          </DialogTrigger>
          <DialogContent>
            <DialogHeader>
              <DialogTitle>Create Profile</DialogTitle>
            </DialogHeader>
            <div className="flex gap-2 mt-2">
              <Input
                placeholder="Profile name (e.g. Wife, Guest)"
                value={newProfileName}
                onChange={(e) => setNewProfileName(e.target.value)}
                onKeyDown={(e) => e.key === "Enter" && handleCreateProfile()}
                autoFocus
              />
              <Button onClick={handleCreateProfile}>Create</Button>
            </div>
          </DialogContent>
        </Dialog>
      </div>

      {/* Profile Cards */}
      <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-3">
        {profiles.map((profile) => (
          <Card
            key={profile.id}
            className="p-5 rounded-2xl shadow-md hover:shadow-lg transition-all duration-200 border relative"
          >
            {!profile.isDefault && (
              <Button
                variant="ghost"
                size="sm"
                className="absolute top-3 right-3 h-7 w-7 p-0 text-muted-foreground hover:text-red-500"
                onClick={() => handleDeleteProfile(profile)}
              >
                <TrashIcon size={14} />
              </Button>
            )}

            <div className="flex items-center gap-2 mb-3">
              <ShieldIcon size={18} className="text-muted-foreground" />
              <h3 className="font-semibold text-lg">{profile.name}</h3>
              {profile.isDefault && (
                <span className="text-xs bg-blue-500/20 text-blue-400 px-2 py-0.5 rounded-full">
                  Default
                </span>
              )}
            </div>

            <Separator className="mb-3" />

            <div className="text-sm text-muted-foreground space-y-1 mb-4">
              <p>
                <span className="text-foreground font-medium">
                  {activeSourcesCount(profile)}
                </span>{" "}
                / {(profile.sources ?? []).length} sources active
              </p>
            </div>

            <Button
              variant="secondary"
              className="w-full text-sm"
              onClick={() => handleOpenProfile(profile)}
            >
              Manage
            </Button>
          </Card>
        ))}
      </div>

      {/* Subnet Rules */}
      <div>
        <div className="flex items-center gap-2 mb-3">
          <NetworkIcon size={18} className="text-muted-foreground" />
          <h2 className="font-semibold text-lg">Subnet Rules</h2>
        </div>
        <p className="text-sm text-muted-foreground mb-4">
          Assign a profile to all clients in a subnet. The most-specific match
          wins.
        </p>

        {/* Add subnet row */}
        <div className="flex gap-2 mb-4 flex-wrap">
          <Input
            placeholder="192.168.30.0/24"
            value={subnetCIDR}
            onChange={(e) => setSubnetCIDR(e.target.value)}
            className="max-w-48"
            onKeyDown={(e) => e.key === "Enter" && handleAddSubnet()}
          />
          <select
            className="flex h-9 rounded-md border border-input bg-transparent px-3 py-1 text-sm shadow-sm focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-ring"
            value={subnetProfileId}
            onChange={(e) =>
              setSubnetProfileId(
                e.target.value === "" ? "" : Number(e.target.value)
              )
            }
          >
            <option value="">Select profile...</option>
            {profiles.map((p) => (
              <option key={p.id} value={p.id}>
                {p.name}
              </option>
            ))}
          </select>
          <Button onClick={handleAddSubnet} disabled={!subnetCIDR || subnetProfileId === ""}>
            <PlusIcon size={16} className="mr-1" />
            Add
          </Button>
        </div>

        {/* Subnet list */}
        {subnets.length === 0 ? (
          <p className="text-sm text-muted-foreground">
            No subnet rules configured. Add one above to route a whole network
            to a profile.
          </p>
        ) : (
          <div className="space-y-2">
            {subnets.map((subnet) => (
              <div
                key={subnet.id}
                className="flex items-center justify-between px-4 py-2 bg-accent rounded-lg"
              >
                <div className="flex items-center gap-4 text-sm">
                  <span className="font-mono font-medium">{subnet.cidr}</span>
                  <span className="text-muted-foreground">→</span>
                  <span>{subnet.profileName}</span>
                </div>
                <Button
                  variant="ghost"
                  size="sm"
                  className="h-7 w-7 p-0 text-muted-foreground hover:text-red-500"
                  onClick={() => handleDeleteSubnet(subnet.id)}
                >
                  <TrashIcon size={14} />
                </Button>
              </div>
            ))}
          </div>
        )}
      </div>

      {/* Profile Detail Sheet */}
      <ProfileDetail
        profile={selectedProfile}
        open={sheetOpen}
        onClose={() => setSheetOpen(false)}
        onRenamed={handleRenamed}
      />
    </div>
  );
}
