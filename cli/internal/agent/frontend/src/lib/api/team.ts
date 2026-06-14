import { BASE, checkedFetch, postJSON } from "./http";

export interface TeamMember {
  name: string;
  aliases?: string[] | null;
  email?: string;
  role?: string;
  department?: string;
  branch?: string;
  registration?: string;
  manager?: string;
}

export async function fetchTeam(): Promise<TeamMember[]> {
  const resp = await checkedFetch(`${BASE}/api/team`);
  return ((await resp.json()) as TeamMember[] | null) ?? [];
}

export function setTeam(members: TeamMember[]): Promise<void> {
  return postJSON("/api/team", members);
}
