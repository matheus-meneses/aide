import { useState } from "react";
import {
  ArrowLeft,
  Boxes,
  Bot,
  IdCard,
  Info,
  Server,
  Settings,
  Store,
  Users,
  type LucideIcon,
} from "lucide-react";
import { cn } from "@/lib/cn";
import { Button, Select } from "@/components/ui";
import { IdentityTab } from "./tabs/IdentityTab";
import { MarketplaceTab } from "./tabs/MarketplaceTab";
import { InstalledTab } from "./tabs/InstalledTab";
import { AgentSettingsTab } from "./tabs/AgentSettingsTab";
import { TeamTab } from "./tabs/TeamTab";
import { GeneralTab } from "./tabs/GeneralTab";
import { RegistriesTab } from "./tabs/RegistriesTab";
import { AboutTab } from "./tabs/AboutTab";

export type TabId =
  | "profile"
  | "marketplace"
  | "installed"
  | "registries"
  | "agent"
  | "team"
  | "general"
  | "about";

type TabDef = { id: TabId; label: string; icon: LucideIcon };

const groups: { label: string; items: TabDef[] }[] = [
  {
    label: "People",
    items: [
      { id: "profile", label: "Profile", icon: IdCard },
      { id: "team", label: "Team", icon: Users },
    ],
  },
  {
    label: "Plugins",
    items: [
      { id: "marketplace", label: "Marketplace", icon: Store },
      { id: "installed", label: "Installed", icon: Boxes },
      { id: "registries", label: "Registries", icon: Server },
    ],
  },
  {
    label: "Agent",
    items: [{ id: "agent", label: "Agent Settings", icon: Bot }],
  },
  {
    label: "System",
    items: [
      { id: "general", label: "General", icon: Settings },
      { id: "about", label: "About", icon: Info },
    ],
  },
];

export function SettingsView({
  onClose,
  initialTab = "profile",
}: {
  onClose: () => void;
  initialTab?: TabId;
}) {
  const [active, setActive] = useState<TabId>(initialTab);
  const [configureTarget, setConfigureTarget] = useState<string>("");

  const goConfigure = (plugin: string) => {
    setConfigureTarget(plugin);
    setActive("installed");
  };

  const renderTab = () => {
    switch (active) {
      case "profile":
        return <IdentityTab />;
      case "team":
        return <TeamTab />;
      case "marketplace":
        return <MarketplaceTab onConfigure={goConfigure} />;
      case "installed":
        return (
          <InstalledTab
            configureTarget={configureTarget}
            onConsumeTarget={() => setConfigureTarget("")}
          />
        );
      case "registries":
        return <RegistriesTab />;
      case "agent":
        return <AgentSettingsTab />;
      case "general":
        return <GeneralTab />;
      case "about":
        return <AboutTab />;
      default:
        return <IdentityTab />;
    }
  };

  return (
    <div className="flex h-full flex-col overflow-hidden">
      <div className="flex items-center gap-2 border-b px-4 py-2">
        <Button variant="ghost" size="icon" onClick={onClose} aria-label="Back to chat">
          <ArrowLeft className="h-4 w-4" />
        </Button>
        <h1 className="text-sm font-semibold">Settings</h1>
      </div>

      <div className="flex min-h-0 flex-1 flex-col overflow-hidden md:flex-row">
        <div className="border-b p-3 md:hidden">
          <Select
            value={active}
            onChange={(e) => setActive(e.target.value as TabId)}
            aria-label="Settings section"
          >
            {groups.map((group) => (
              <optgroup key={group.label} label={group.label}>
                {group.items.map((t) => (
                  <option key={t.id} value={t.id}>
                    {t.label}
                  </option>
                ))}
              </optgroup>
            ))}
          </Select>
        </div>

        <nav className="hidden w-48 shrink-0 space-y-4 overflow-y-auto border-r p-3 md:block">
          {groups.map((group) => (
            <div key={group.label} className="space-y-1">
              <div className="px-2.5 text-[11px] font-medium uppercase tracking-wider text-muted-foreground/70">
                {group.label}
              </div>
              {group.items.map((t) => {
                const Icon = t.icon;
                return (
                  <button
                    key={t.id}
                    onClick={() => setActive(t.id)}
                    className={cn(
                      "flex w-full items-center gap-2 rounded-md px-2.5 py-2 text-sm transition-colors",
                      active === t.id
                        ? "bg-accent font-medium text-accent-foreground"
                        : "text-muted-foreground hover:bg-accent/60 hover:text-foreground",
                    )}
                    aria-current={active === t.id}
                  >
                    <Icon className="h-4 w-4 shrink-0" />
                    {t.label}
                  </button>
                );
              })}
            </div>
          ))}
        </nav>

        <div className="min-h-0 flex-1 overflow-y-auto p-4 md:p-5">
          <div key={active} className="mx-auto max-w-2xl animate-fade-in">
            {renderTab()}
          </div>
        </div>
      </div>
    </div>
  );
}
