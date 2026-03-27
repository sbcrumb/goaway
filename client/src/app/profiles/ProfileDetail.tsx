import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Separator } from "@/components/ui/separator";
import {
  Sheet,
  SheetContent,
  SheetHeader,
  SheetTitle
} from "@/components/ui/sheet";
import { Switch } from "@/components/ui/switch";
import {
  Tabs,
  TabsContent,
  TabsList,
  TabsTrigger
} from "@/components/ui/tabs";
import { DeleteRequest, GetRequest, PostRequest, PutRequest } from "@/util";
import {
  ArrowsClockwiseIcon,
  CheckIcon,
  PencilIcon,
  TrashIcon,
  XIcon
} from "@phosphor-icons/react";
import { useEffect, useState } from "react";
import { toast } from "sonner";

export type ProfileSourceStatus = {
  sourceId: number;
  name: string;
  url: string;
  active: boolean;
};

export type ProfileDetail = {
  id: number;
  name: string;
  isDefault: boolean;
  sources: ProfileSourceStatus[];
};

interface Props {
  profile: ProfileDetail | null;
  open: boolean;
  onClose: () => void;
  onRenamed: (id: number, name: string) => void;
}

export function ProfileDetail({ profile, open, onClose, onRenamed }: Props) {
  const [sources, setSources] = useState<ProfileSourceStatus[]>([]);
  const [blacklist, setBlacklist] = useState<string[]>([]);
  const [whitelist, setWhitelist] = useState<string[]>([]);
  const [newBlacklist, setNewBlacklist] = useState("");
  const [newWhitelist, setNewWhitelist] = useState("");
  const [isEditing, setIsEditing] = useState(false);
  const [editedName, setEditedName] = useState("");
  const [saving, setSaving] = useState(false);

  useEffect(() => {
    if (!profile || !open) return;
    setEditedName(profile.name);
    setSources(profile.sources ?? []);
    fetchBlacklist();
    fetchWhitelist();
  }, [profile, open]);

  const fetchBlacklist = async () => {
    if (!profile) return;
    const [code, data] = await GetRequest(`profiles/${profile.id}/blacklist`);
    if (code === 200) setBlacklist(data.domains ?? []);
  };

  const fetchWhitelist = async () => {
    if (!profile) return;
    const [code, data] = await GetRequest(`profiles/${profile.id}/whitelist`);
    if (code === 200) setWhitelist(data.domains ?? []);
  };

  const handleToggleSource = async (sourceId: number, active: boolean) => {
    if (!profile) return;
    const [code] = await PutRequest(
      `profiles/${profile.id}/sources/${sourceId}/toggle`,
      { active }
    );
    if (code === 200) {
      setSources((prev) =>
        prev.map((s) => (s.sourceId === sourceId ? { ...s, active } : s))
      );
    } else {
      toast.error("Failed to toggle source");
    }
  };

  const handleAddBlacklist = async () => {
    const domains = newBlacklist
      .split(/[\n,]+/)
      .map((d) => d.trim())
      .filter(Boolean);
    if (!domains.length || !profile) return;
    const [code] = await PostRequest(`profiles/${profile.id}/blacklist`, {
      domains
    });
    if (code === 201) {
      setBlacklist((prev) => [...new Set([...prev, ...domains])]);
      setNewBlacklist("");
      toast.success(`Added ${domains.length} domain(s) to blacklist`);
    }
  };

  const handleRemoveBlacklist = async (domain: string) => {
    if (!profile) return;
    const [code] = await DeleteRequest(
      `profiles/${profile.id}/blacklist?domain=${encodeURIComponent(domain)}`,
      null
    );
    if (code === 200) {
      setBlacklist((prev) => prev.filter((d) => d !== domain));
    }
  };

  const handleAddWhitelist = async () => {
    const domain = newWhitelist.trim();
    if (!domain || !profile) return;
    const [code] = await PostRequest(`profiles/${profile.id}/whitelist`, {
      domain
    });
    if (code === 201) {
      setWhitelist((prev) => [...new Set([...prev, domain])]);
      setNewWhitelist("");
      toast.success(`Added ${domain} to whitelist`);
    }
  };

  const handleRemoveWhitelist = async (domain: string) => {
    if (!profile) return;
    const [code] = await DeleteRequest(
      `profiles/${profile.id}/whitelist?domain=${encodeURIComponent(domain)}`,
      null
    );
    if (code === 200) {
      setWhitelist((prev) => prev.filter((d) => d !== domain));
    }
  };

  const handleRename = async () => {
    if (!profile || editedName.trim() === "") return;
    if (editedName.trim() === profile.name) {
      setIsEditing(false);
      return;
    }
    setSaving(true);
    const [code] = await PutRequest(`profiles/${profile.id}/name`, {
      name: editedName.trim()
    });
    setSaving(false);
    if (code === 200) {
      onRenamed(profile.id, editedName.trim());
      setIsEditing(false);
      toast.success(`Profile renamed to "${editedName.trim()}"`);
    } else {
      toast.error("Failed to rename profile");
    }
  };

  if (!profile) return null;

  return (
    <Sheet open={open} onOpenChange={(v) => !v && onClose()}>
      <SheetContent className="w-full sm:max-w-lg overflow-y-auto">
        <SheetHeader className="mb-4">
          <SheetTitle className="flex items-center gap-2">
            {isEditing ? (
              <div className="flex items-center gap-2 flex-1">
                <Input
                  value={editedName}
                  onChange={(e) => setEditedName(e.target.value)}
                  className="flex-1"
                  autoFocus
                  disabled={saving}
                />
                <Button
                  size="sm"
                  variant="ghost"
                  onClick={handleRename}
                  disabled={saving}
                  className="h-8 w-8 p-0"
                >
                  {saving ? (
                    <ArrowsClockwiseIcon className="animate-spin" size={16} />
                  ) : (
                    <CheckIcon className="text-green-600" size={16} />
                  )}
                </Button>
                <Button
                  size="sm"
                  variant="ghost"
                  onClick={() => {
                    setEditedName(profile.name);
                    setIsEditing(false);
                  }}
                  disabled={saving}
                  className="h-8 w-8 p-0"
                >
                  <XIcon className="text-red-600" size={16} />
                </Button>
              </div>
            ) : (
              <>
                <span>{profile.name}</span>
                {profile.isDefault && (
                  <span className="text-xs bg-blue-500/20 text-blue-400 px-2 py-0.5 rounded-full">
                    Default
                  </span>
                )}
                {!profile.isDefault && (
                  <Button
                    size="sm"
                    variant="ghost"
                    onClick={() => setIsEditing(true)}
                    className="h-8 w-8 p-0"
                  >
                    <PencilIcon className="text-muted-foreground" size={16} />
                  </Button>
                )}
              </>
            )}
          </SheetTitle>
        </SheetHeader>

        <Tabs defaultValue="sources">
          <TabsList className="w-full">
            <TabsTrigger value="sources" className="flex-1">
              Sources
            </TabsTrigger>
            <TabsTrigger value="blacklist" className="flex-1">
              Blacklist
            </TabsTrigger>
            <TabsTrigger value="whitelist" className="flex-1">
              Whitelist
            </TabsTrigger>
          </TabsList>

          {/* Sources Tab */}
          <TabsContent value="sources" className="mt-4 space-y-2">
            {sources.length === 0 ? (
              <p className="text-sm text-muted-foreground text-center py-8">
                No sources available.
              </p>
            ) : (
              sources.map((source) => (
                <div
                  key={source.sourceId}
                  className="flex items-center justify-between p-3 bg-accent rounded-lg"
                >
                  <div className="flex-1 min-w-0 mr-3">
                    <p className="text-sm font-medium truncate">{source.name}</p>
                    <p className="text-xs text-muted-foreground truncate">
                      {source.url}
                    </p>
                  </div>
                  <Switch
                    checked={source.active}
                    onCheckedChange={(checked) =>
                      handleToggleSource(source.sourceId, checked)
                    }
                  />
                </div>
              ))
            )}
          </TabsContent>

          {/* Blacklist Tab */}
          <TabsContent value="blacklist" className="mt-4 space-y-3">
            <div className="flex gap-2">
              <Input
                placeholder="domain.com (comma or newline separated)"
                value={newBlacklist}
                onChange={(e) => setNewBlacklist(e.target.value)}
                onKeyDown={(e) => e.key === "Enter" && handleAddBlacklist()}
              />
              <Button onClick={handleAddBlacklist} size="sm">
                Add
              </Button>
            </div>
            <Separator />
            <div className="space-y-1 max-h-80 overflow-y-auto">
              {blacklist.length === 0 ? (
                <p className="text-sm text-muted-foreground text-center py-4">
                  No custom blocked domains.
                </p>
              ) : (
                blacklist.map((domain) => (
                  <div
                    key={domain}
                    className="flex items-center justify-between px-3 py-1.5 bg-accent rounded text-sm"
                  >
                    <span className="truncate">{domain}</span>
                    <Button
                      variant="ghost"
                      size="sm"
                      className="h-6 w-6 p-0 shrink-0 ml-2"
                      onClick={() => handleRemoveBlacklist(domain)}
                    >
                      <TrashIcon size={12} className="text-red-500" />
                    </Button>
                  </div>
                ))
              )}
            </div>
          </TabsContent>

          {/* Whitelist Tab */}
          <TabsContent value="whitelist" className="mt-4 space-y-3">
            <div className="flex gap-2">
              <Input
                placeholder="domain.com"
                value={newWhitelist}
                onChange={(e) => setNewWhitelist(e.target.value)}
                onKeyDown={(e) => e.key === "Enter" && handleAddWhitelist()}
              />
              <Button onClick={handleAddWhitelist} size="sm">
                Add
              </Button>
            </div>
            <Separator />
            <div className="space-y-1 max-h-80 overflow-y-auto">
              {whitelist.length === 0 ? (
                <p className="text-sm text-muted-foreground text-center py-4">
                  No custom allowed domains.
                </p>
              ) : (
                whitelist.map((domain) => (
                  <div
                    key={domain}
                    className="flex items-center justify-between px-3 py-1.5 bg-accent rounded text-sm"
                  >
                    <span className="truncate">{domain}</span>
                    <Button
                      variant="ghost"
                      size="sm"
                      className="h-6 w-6 p-0 shrink-0 ml-2"
                      onClick={() => handleRemoveWhitelist(domain)}
                    >
                      <TrashIcon size={12} className="text-red-500" />
                    </Button>
                  </div>
                ))
              )}
            </div>
          </TabsContent>
        </Tabs>
      </SheetContent>
    </Sheet>
  );
}
