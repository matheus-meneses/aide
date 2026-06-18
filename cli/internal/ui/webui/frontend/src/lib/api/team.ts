import { BASE, checkedFetch } from "./http";

export interface TeamMember {
  id: number;
  name: string;
  email?: string;
  aliases?: string[] | null;
  role?: string;
  department?: string;
  branch?: string;
  registration?: string;
  manager_id?: number | null;
  manager_registration?: string;
  source: string;
}

export interface TeamMemberInput {
  name: string;
  email?: string;
  aliases?: string[];
  role?: string;
  department?: string;
  branch?: string;
  registration?: string;
  manager_id?: number | null;
  manager_registration?: string;
}

export async function fetchTeam(): Promise<TeamMember[]> {
  const resp = await checkedFetch(`${BASE}/api/team`);
  return ((await resp.json()) as TeamMember[] | null) ?? [];
}

export async function addTeamMember(member: TeamMemberInput): Promise<TeamMember> {
  const resp = await checkedFetch(`${BASE}/api/team`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(member),
  });
  return (await resp.json()) as TeamMember;
}

export async function updateTeamMember(id: number, member: TeamMemberInput): Promise<void> {
  await checkedFetch(`${BASE}/api/team/${id}`, {
    method: "PUT",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(member),
  });
}

export async function deleteTeamMember(id: number): Promise<void> {
  await checkedFetch(`${BASE}/api/team/${id}`, { method: "DELETE" });
}
