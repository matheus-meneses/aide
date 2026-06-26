import { AIModelTab } from "./AIModelTab";
import { ContextTab } from "./ContextTab";
import { ScheduleTab } from "./ScheduleTab";

export function AgentSettingsTab() {
  return (
    <div className="space-y-8">
      <AIModelTab />
      <div className="border-t" />
      <ContextTab />
      <div className="border-t" />
      <ScheduleTab />
    </div>
  );
}
