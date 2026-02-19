import { useEffect, useMemo, useState } from "react";
import { SafeAreaView, ScrollView, Text, TextInput, TouchableOpacity, View } from "react-native";
import { ROUTE_TEMPLATES, fillRoute } from "@raidx/shared";
import { authClient } from "./authClient";
import { Button, TabRow, SectionCard, StatusBox, ListItem } from "./ui";
import {
  normalizeRole,
  normalizeStatus,
  getInviteId,
  getDeclineReason,
  getStatusDisplay,
  safeArray,
  getTeamId,
  getObjectIdString
} from "./helpers";

function PlayerPanel() {
  const [inviteTab, setInviteTab] = useState("pending");
  const [infoTab, setInfoTab] = useState("teams");
  const [invites, setInvites] = useState({ pending: [], accepted: [], declined: [] });
  const [teams, setTeams] = useState([]);
  const [events, setEvents] = useState([]);
  const [message, setMessage] = useState("");
  const [messageType, setMessageType] = useState("info");

  async function loadData() {
    setMessage("");
    try {
      const [invitesRes, teamsRes, eventsRes] = await Promise.all([
        authClient.apiFetch("/api/invitations"),
        authClient.apiFetch("/api/player/teams"),
        authClient.apiFetch("/api/player/events")
      ]);

      const allInvites = safeArray(await invitesRes.json());
      setInvites({
        pending: allInvites.filter((invite) => ["pending", "invited_via_link"].includes(normalizeStatus(invite))),
        accepted: allInvites.filter((invite) => ["accepted", "accepted_by_owner"].includes(normalizeStatus(invite))),
        declined: allInvites.filter((invite) => ["declined", "declined_by_owner"].includes(normalizeStatus(invite)))
      });
      setTeams(safeArray(await teamsRes.json()));
      setEvents(safeArray(await eventsRes.json()));
    } catch (error) {
      setMessageType("danger");
      setMessage(error.message || "Failed to load player data");
    }
  }

  useEffect(() => {
    loadData();
  }, []);

  async function updateInvite(inviteId, status) {
    try {
      const response = await authClient.apiFetch(`/api/invitations/${inviteId}`, {
        method: "PUT",
        body: JSON.stringify({ status })
      });
      const data = await response.json().catch(() => ({}));
      if (!response.ok) throw new Error(data.error || "Failed to update invitation");
      await loadData();
    } catch (error) {
      setMessageType("danger");
      setMessage(error.message || "Failed to update invitation");
    }
  }

  const currentInvites = invites[inviteTab] || [];

  return (
    <>
      <SectionCard title="Team Invitations" subtitle="Review and manage incoming team requests.">
        <StatusBox message={message} type={messageType} />
        <Button label="Refresh" variant="primary" onPress={loadData} />
        <TabRow
          current={inviteTab}
          onChange={setInviteTab}
          items={[
            { value: "pending", label: `Pending (${invites.pending.length})` },
            { value: "accepted", label: `Accepted (${invites.accepted.length})` },
            { value: "declined", label: `Declined (${invites.declined.length})` }
          ]}
        />
        {!currentInvites.length ? <Text style={{ color: "#94a3b8" }}>No invitations in this tab.</Text> : null}
        {currentInvites.map((invite) => (
          <ListItem key={getInviteId(invite)} title={invite.teamName || "Team invite"}>
            <Text style={{ color: "#cbd5e1" }}>Owner: {invite.ownerName || "Unknown"}</Text>
            <Text style={{ color: "#cbd5e1" }}>Status: {getStatusDisplay(normalizeStatus(invite))}</Text>
            {getDeclineReason(invite) ? <Text style={{ color: "#cbd5e1" }}>Reason: {getDeclineReason(invite)}</Text> : null}
            {inviteTab === "pending" && normalizeStatus(invite) !== "invited_via_link" ? (
              <View style={{ marginTop: 8 }}>
                <Button label="Accept" variant="primary" onPress={() => updateInvite(getInviteId(invite), "accepted")} />
                <Button label="Decline" variant="danger" onPress={() => updateInvite(getInviteId(invite), "declined")} />
              </View>
            ) : null}
          </ListItem>
        ))}
      </SectionCard>

      <SectionCard title="My Teams & Events" subtitle="Teams you belong to and recent events.">
        <Button label="Refresh" variant="primary" onPress={loadData} />
        <TabRow
          current={infoTab}
          onChange={setInfoTab}
          items={[
            { value: "teams", label: `My Teams (${teams.length})` },
            { value: "events", label: `My Events (${events.length})` }
          ]}
        />

        {infoTab === "teams" ? (
          <>
            {!teams.length ? <Text style={{ color: "#94a3b8" }}>No teams yet.</Text> : null}
            {teams.map((team) => (
              <ListItem key={team.teamId || team.TeamID || team.ID} title={team.teamName || team.TeamName || "Team"}>
                <Text style={{ color: "#cbd5e1" }}>Status: {team.status || team.Status || "active"}</Text>
              </ListItem>
            ))}
          </>
        ) : (
          <>
            {!events.length ? <Text style={{ color: "#94a3b8" }}>No events yet.</Text> : null}
            {events.map((event) => (
              <ListItem key={`${event.eventType}-${event.eventId || event.eventName}`} title={event.eventName || "Event"}>
                <Text style={{ color: "#cbd5e1" }}>Type: {event.eventType || "event"}</Text>
                <Text style={{ color: "#cbd5e1" }}>Matches: {event.matchCount || 0}</Text>
              </ListItem>
            ))}
          </>
        )}
      </SectionCard>
    </>
  );
}

function OwnerPanel() {
  const [tab, setTab] = useState("teams");
  const [inviteTab, setInviteTab] = useState("pending");
  const [teams, setTeams] = useState([]);
  const [invites, setInvites] = useState({ pending: [], accepted: [], declined: [] });
  const [selectedTeamByInvite, setSelectedTeamByInvite] = useState({});
  const [selectedTeamId, setSelectedTeamId] = useState("");
  const [teamDetail, setTeamDetail] = useState(null);
  const [teamEditName, setTeamEditName] = useState("");
  const [teamEditStatus, setTeamEditStatus] = useState("active");
  const [pendingApprovals, setPendingApprovals] = useState([]);
  const [inviteLinks, setInviteLinks] = useState([]);
  const [inviteTeamId, setInviteTeamId] = useState("");
  const [invitePlayerIdentifier, setInvitePlayerIdentifier] = useState("");
  const [teamInvites, setTeamInvites] = useState([]);
  const [tournamentRequests, setTournamentRequests] = useState([]);
  const [lookupTeamId, setLookupTeamId] = useState("");
  const [lookupTeamData, setLookupTeamData] = useState(null);
  const [matchLookupId, setMatchLookupId] = useState("");
  const [raidType, setRaidType] = useState("successful");
  const [raiderId, setRaiderId] = useState("");
  const [defenderIds, setDefenderIds] = useState("");
  const [bonusTaken, setBonusTaken] = useState(false);
  const [generatedTeamLink, setGeneratedTeamLink] = useState("");
  const [playerIdentifier, setPlayerIdentifier] = useState("");
  const [linkExpiresIn, setLinkExpiresIn] = useState("30d");
  const [linkMaxUses, setLinkMaxUses] = useState("");
  const [teamName, setTeamName] = useState("");
  const [message, setMessage] = useState("");
  const [messageType, setMessageType] = useState("info");

  async function loadData() {
    setMessage("");
    try {
      const [teamsRes, invitesRes] = await Promise.all([
        authClient.apiFetch("/api/owner/teams"),
        authClient.apiFetch("/api/owner/event-invitations")
      ]);
      const teamData = safeArray(await teamsRes.json());
      const inviteData = safeArray(await invitesRes.json());
      setTeams(teamData);
      setInvites({
        pending: inviteData.filter((invite) => ["pending", "invited_via_link"].includes(normalizeStatus(invite))),
        accepted: inviteData.filter((invite) => ["accepted", "accepted_by_owner", "accepted_by_organizer"].includes(normalizeStatus(invite))),
        declined: inviteData.filter((invite) => ["declined", "declined_by_organizer"].includes(normalizeStatus(invite)))
      });
      if (!selectedTeamId && teamData.length > 0) {
        setSelectedTeamId(getTeamId(teamData[0]));
      }
    } catch (error) {
      setMessageType("danger");
      setMessage(error.message || "Failed to load owner data");
    }
  }

  async function loadTeamContext(teamId) {
    if (!teamId) {
      setTeamDetail(null);
      setPendingApprovals([]);
      setInviteLinks([]);
      return;
    }

    try {
      const [teamRes, approvalsRes, linksRes] = await Promise.all([
        authClient.apiFetch(`/api/teams/${teamId}`),
        authClient.apiFetch(`/api/teams/${teamId}/pending-approvals`),
        authClient.apiFetch("/api/owner/invite-links")
      ]);
      const teamData = await teamRes.json().catch(() => ({}));
      const approvalsData = await approvalsRes.json().catch(() => []);
      const linksData = await linksRes.json().catch(() => ({}));

      if (!teamRes.ok) throw new Error(teamData.error || "Failed to load team details");
      if (!approvalsRes.ok) throw new Error(approvalsData.error || "Failed to load pending approvals");
      if (!linksRes.ok) throw new Error(linksData.error || "Failed to load invite links");

      setTeamDetail(teamData);
      setTeamEditName(teamData.teamName || "");
      setTeamEditStatus(teamData.status || "active");
      setPendingApprovals(safeArray(approvalsData));
      setInviteLinks(safeArray(linksData.data).filter((link) => (link.targetId || "") === teamId));
    } catch (error) {
      setMessageType("danger");
      setMessage(error.message || "Failed to load team context");
    }
  }

  useEffect(() => {
    loadData();
  }, []);

  useEffect(() => {
    if (!selectedTeamId) return;
    loadTeamContext(selectedTeamId);
  }, [selectedTeamId]);

  async function createTeam() {
    if (!teamName.trim()) {
      setMessageType("warning");
      setMessage("Team name is required");
      return;
    }
    try {
      const response = await authClient.apiFetch("/api/teams", {
        method: "POST",
        body: JSON.stringify({ team_name: teamName.trim() })
      });
      const data = await response.json().catch(() => ({}));
      if (!response.ok) throw new Error(data.error || "Failed to create team");
      setTeamName("");
      setTab("teams");
      setMessageType("success");
      setMessage(`Team created: ${data.team_name || "new team"}`);
      await loadData();
    } catch (error) {
      setMessageType("danger");
      setMessage(error.message || "Failed to create team");
    }
  }

  async function updateInvite(invite, status) {
    const inviteId = getInviteId(invite);
    const payload = { status };
    if (status === "accepted") {
      const selectedTeamId = selectedTeamByInvite[inviteId] || invite.teamId || "";
      if (!selectedTeamId) {
        setMessageType("warning");
        setMessage("Select a team before accepting.");
        return;
      }
      const selectedTeam = teams.find((team) => getTeamId(team) === selectedTeamId);
      const playersCount = Number(selectedTeam?.Players || selectedTeam?.players || 0);
      if (playersCount < 7) {
        setMessageType("warning");
        setMessage("Selected team must have at least 7 players.");
        return;
      }
      payload.team_id = selectedTeamId;
    }

    try {
      const response = await authClient.apiFetch(`/api/invitations/${inviteId}`, {
        method: "PUT",
        body: JSON.stringify(payload)
      });
      const data = await response.json().catch(() => ({}));
      if (!response.ok) throw new Error(data.error || "Failed to update invitation");
      setMessageType("success");
      setMessage("Invitation updated");
      await loadData();
    } catch (error) {
      setMessageType("danger");
      setMessage(error.message || "Failed to update invitation");
    }
  }

  async function updateTeamDetails() {
    if (!selectedTeamId) return;
    try {
      const response = await authClient.apiFetch(`/api/teams/${selectedTeamId}`, {
        method: "PUT",
        body: JSON.stringify({ team_name: teamEditName, status: teamEditStatus })
      });
      const data = await response.json().catch(() => ({}));
      if (!response.ok) throw new Error(data.error || "Failed to update team");
      setMessageType("success");
      setMessage("Team updated");
      await loadData();
      await loadTeamContext(selectedTeamId);
    } catch (error) {
      setMessageType("danger");
      setMessage(error.message || "Failed to update team");
    }
  }

  async function addPlayerToTeam() {
    if (!selectedTeamId || !playerIdentifier.trim()) return;
    try {
      const response = await authClient.apiFetch(`/api/teams/${selectedTeamId}/add-player`, {
        method: "POST",
        body: JSON.stringify({ playerIdentifier: playerIdentifier.trim() })
      });
      const data = await response.json().catch(() => ({}));
      if (!response.ok) throw new Error(data.error || "Failed to add player");
      setPlayerIdentifier("");
      setMessageType("success");
      setMessage("Player added");
      await loadData();
      await loadTeamContext(selectedTeamId);
    } catch (error) {
      setMessageType("danger");
      setMessage(error.message || "Failed to add player");
    }
  }

  async function removePlayerFromTeam(playerId) {
    try {
      const response = await authClient.apiFetch(`/api/teams/${selectedTeamId}/remove-player/${playerId}`, {
        method: "DELETE"
      });
      const data = await response.json().catch(() => ({}));
      if (!response.ok) throw new Error(data.error || "Failed to remove player");
      setMessageType("success");
      setMessage("Player removed");
      await loadData();
      await loadTeamContext(selectedTeamId);
    } catch (error) {
      setMessageType("danger");
      setMessage(error.message || "Failed to remove player");
    }
  }

  async function deleteTeam() {
    if (!selectedTeamId) return;
    try {
      const response = await authClient.apiFetch(`/api/teams/${selectedTeamId}`, { method: "DELETE" });
      const data = await response.json().catch(() => ({}));
      if (!response.ok) throw new Error(data.error || "Failed to delete team");
      setSelectedTeamId("");
      setTeamDetail(null);
      setPendingApprovals([]);
      setInviteLinks([]);
      setMessageType("success");
      setMessage("Team deleted");
      await loadData();
    } catch (error) {
      setMessageType("danger");
      setMessage(error.message || "Failed to delete team");
    }
  }

  async function handleOwnerApproval(approvalId, action) {
    const route = action === "approve"
      ? fillRoute(ROUTE_TEMPLATES.pendingApprovalApprove, { id: approvalId })
      : fillRoute(ROUTE_TEMPLATES.pendingApprovalReject, { id: approvalId });
    try {
      const response = await authClient.apiFetch(route, { method: "PUT" });
      const data = await response.json().catch(() => ({}));
      if (!response.ok) throw new Error(data.error || `Failed to ${action} approval`);
      setMessageType("success");
      setMessage(data.message || `Approval ${action}d`);
      await loadTeamContext(selectedTeamId);
      await loadData();
    } catch (error) {
      setMessageType("danger");
      setMessage(error.message || `Failed to ${action} approval`);
    }
  }

  async function createOwnerInviteLink() {
    if (!selectedTeamId) return;
    const payload = {
      targetId: selectedTeamId,
      expiresIn: linkExpiresIn
    };
    if (Number(linkMaxUses) > 0) payload.maxUses = Number(linkMaxUses);
    try {
      const response = await authClient.apiFetch("/api/owner/invite-links", {
        method: "POST",
        body: JSON.stringify(payload)
      });
      const data = await response.json().catch(() => ({}));
      if (!response.ok) throw new Error(data.error || "Failed to create invite link");
      setLinkMaxUses("");
      setMessageType("success");
      setMessage("Invite link created");
      await loadTeamContext(selectedTeamId);
    } catch (error) {
      setMessageType("danger");
      setMessage(error.message || "Failed to create invite link");
    }
  }

  async function deleteOwnerInviteLink(linkId) {
    try {
      const response = await authClient.apiFetch(`/api/owner/invite-links/${linkId}`, { method: "DELETE" });
      const data = await response.json().catch(() => ({}));
      if (!response.ok) throw new Error(data.error || "Failed to delete invite link");
      setMessageType("success");
      setMessage("Invite link deleted");
      await loadTeamContext(selectedTeamId);
    } catch (error) {
      setMessageType("danger");
      setMessage(error.message || "Failed to delete invite link");
    }
  }

  async function createTeamInvite() {
    if (!inviteTeamId || !invitePlayerIdentifier.trim()) return;
    try {
      const response = await authClient.apiFetch(`/api/teams/${inviteTeamId}/invite`, {
        method: "POST",
        body: JSON.stringify({ playerIdentifier: invitePlayerIdentifier.trim() })
      });
      const data = await response.json().catch(() => ({}));
      if (!response.ok) throw new Error(data.error || "Failed to create team invite");
      setInvitePlayerIdentifier("");
      setMessageType("success");
      setMessage("Team invite created");
      await loadTeamInvites(inviteTeamId);
    } catch (error) {
      setMessageType("danger");
      setMessage(error.message || "Failed to create team invite");
    }
  }

  async function loadTeamInvites(teamId = inviteTeamId) {
    if (!teamId) return;
    try {
      const response = await authClient.apiFetch(`/api/teams/${teamId}/invites`);
      const data = await response.json().catch(() => []);
      if (!response.ok) throw new Error(data.error || "Failed to load team invites");
      setTeamInvites(safeArray(data));
    } catch (error) {
      setMessageType("danger");
      setMessage(error.message || "Failed to load team invites");
    }
  }

  async function loadOwnerTournamentRequests() {
    try {
      const response = await authClient.apiFetch("/api/owner/tournament-requests");
      const data = await response.json().catch(() => []);
      if (!response.ok) throw new Error(data.error || "Failed to load tournament requests");
      setTournamentRequests(safeArray(data));
    } catch (error) {
      setMessageType("danger");
      setMessage(error.message || "Failed to load tournament requests");
    }
  }

  async function lookupTeamAlias() {
    if (!lookupTeamId.trim()) return;
    try {
      const response = await authClient.apiFetch(`/api/team/${lookupTeamId.trim()}`);
      const data = await response.json().catch(() => ({}));
      if (!response.ok) throw new Error(data.error || "Failed to fetch /api/team/:id");
      setLookupTeamData(data);
    } catch (error) {
      setMessageType("danger");
      setMessage(error.message || "Failed to lookup team alias route");
    }
  }

  async function openMatchRoute() {
    if (!matchLookupId.trim()) return;
    try {
      const response = await authClient.apiFetch(`/api/matches/${matchLookupId.trim()}`);
      const html = await response.text();
      if (!response.ok) throw new Error("Failed to open /api/matches/:id route");
      setMessageType("info");
      setMessage(`Route opened, HTML length: ${html.length}`);
    } catch (error) {
      setMessageType("danger");
      setMessage(error.message || "Failed to open match route");
    }
  }

  async function openMatchesRoute() {
    try {
      const response = await authClient.apiFetch("/api/matches");
      const html = await response.text();
      if (!response.ok) throw new Error("Failed to open /api/matches route");
      setMessageType("info");
      setMessage(`Matches route opened, HTML length: ${html.length}`);
    } catch (error) {
      setMessageType("danger");
      setMessage(error.message || "Failed to open matches route");
    }
  }

  async function callEndgameApi() {
    try {
      const response = await authClient.apiFetch("/api/endgame");
      const body = await response.text();
      if (!response.ok) throw new Error("Failed to call /api/endgame");
      setMessageType("info");
      setMessage(`/api/endgame response length: ${body.length}`);
    } catch (error) {
      setMessageType("danger");
      setMessage(error.message || "Failed to call /api/endgame");
    }
  }

  async function submitRaidResult() {
    if (!raiderId.trim()) return;
    const payload = {
      raidType,
      raiderId: raiderId.trim(),
      defenderIds: defenderIds.split(",").map((item) => item.trim()).filter(Boolean),
      bonusTaken
    };
    try {
      const response = await authClient.apiFetch("/api/matches/raid", {
        method: "POST",
        body: JSON.stringify(payload)
      });
      const data = await response.json().catch(() => ({}));
      if (!response.ok) throw new Error(data.error || "Failed to process raid");
      setMessageType("success");
      setMessage("Raid processed successfully");
    } catch (error) {
      setMessageType("danger");
      setMessage(error.message || "Failed to process raid");
    }
  }

  async function generateLegacyTeamLink() {
    if (!selectedTeamId) return;
    try {
      const response = await authClient.apiFetch(`/api/teams/${selectedTeamId}/generate-link`, { method: "POST" });
      const data = await response.json().catch(() => ({}));
      if (!response.ok) throw new Error(data.error || "Failed to generate team invite link");
      setGeneratedTeamLink(data.invite_url || (data.invite_token ? `/invite/team/${data.invite_token}` : ""));
      setMessageType("success");
      setMessage("Legacy team invite link generated");
    } catch (error) {
      setMessageType("danger");
      setMessage(error.message || "Failed to generate team invite link");
    }
  }

  const selectedInvites = invites[inviteTab] || [];

  return (
    <SectionCard title="Team Owner" subtitle="Manage your teams and event invitations.">
      <StatusBox message={message} type={messageType} />
      <Button label="Refresh" variant="primary" onPress={loadData} />
      <TabRow
        current={tab}
        onChange={setTab}
        items={[
          { value: "teams", label: `My Teams (${teams.length})` },
          { value: "invites", label: "Event Invitations" },
          { value: "manage", label: "Team Manager" },
          { value: "approvals", label: `Pending Approvals (${pendingApprovals.length})` },
          { value: "links", label: `Invite Links (${inviteLinks.length})` },
          { value: "advanced", label: "Advanced APIs" },
          { value: "create", label: "Create Team" }
        ]}
      />

      {tab === "teams" ? (
        <>
          {!teams.length ? <Text style={{ color: "#94a3b8" }}>No teams found.</Text> : null}
          {teams.map((team) => (
            <ListItem key={getTeamId(team)} title={team.TeamName || team.teamName || "Team"}>
              <Text style={{ color: "#cbd5e1" }}>Players: {Number(team.Players || team.players || 0)}</Text>
              <Text style={{ color: "#cbd5e1" }}>Status: {team.Status || team.status || "active"}</Text>
            </ListItem>
          ))}
        </>
      ) : null}

      {tab === "invites" ? (
        <>
          <TabRow
            current={inviteTab}
            onChange={setInviteTab}
            items={[
              { value: "pending", label: `Pending (${invites.pending.length})` },
              { value: "accepted", label: `Accepted (${invites.accepted.length})` },
              { value: "declined", label: `Declined (${invites.declined.length})` }
            ]}
          />
          {!selectedInvites.length ? <Text style={{ color: "#94a3b8" }}>No invitations in this tab.</Text> : null}
          {selectedInvites.map((invite) => (
            <ListItem key={getInviteId(invite)} title={invite.eventName || "Event Invite"}>
              <Text style={{ color: "#cbd5e1" }}>Owner: {invite.ownerName || "Unknown"}</Text>
              <Text style={{ color: "#cbd5e1" }}>Team: {invite.teamName || "Unassigned"}</Text>
              <Text style={{ color: "#cbd5e1" }}>Status: {getStatusDisplay(normalizeStatus(invite))}</Text>
              {getDeclineReason(invite) ? <Text style={{ color: "#cbd5e1" }}>Reason: {getDeclineReason(invite)}</Text> : null}

              {inviteTab === "pending" && normalizeStatus(invite) !== "invited_via_link" ? (
                <View style={{ marginTop: 8 }}>
                  <Text style={{ color: "#e2e8f0", marginBottom: 6 }}>Select Team:</Text>
                  <View style={{ flexDirection: "row", flexWrap: "wrap", gap: 8, marginBottom: 8 }}>
                    {teams.map((team) => {
                      const teamId = getTeamId(team);
                      const playersCount = Number(team.Players || team.players || 0);
                      const selected = selectedTeamByInvite[getInviteId(invite)] === teamId;
                      return (
                        <TouchableOpacity
                          key={`${getInviteId(invite)}-${teamId}`}
                          disabled={playersCount < 7}
                          onPress={() => setSelectedTeamByInvite((prev) => ({ ...prev, [getInviteId(invite)]: teamId }))}
                          style={{
                            paddingVertical: 6,
                            paddingHorizontal: 10,
                            borderRadius: 999,
                            backgroundColor: selected ? "#f97316" : "#334155",
                            opacity: playersCount < 7 ? 0.45 : 1
                          }}
                        >
                          <Text style={{ color: "white" }}>{team.TeamName || team.teamName} ({playersCount})</Text>
                        </TouchableOpacity>
                      );
                    })}
                  </View>
                  <Button label="Accept" variant="primary" onPress={() => updateInvite(invite, "accepted")} />
                  <Button label="Decline" variant="danger" onPress={() => updateInvite(invite, "declined")} />
                </View>
              ) : null}
            </ListItem>
          ))}
        </>
      ) : null}

      {tab === "create" ? (
        <View>
          <TextInput
            placeholder="Team name"
            placeholderTextColor="#94a3b8"
            value={teamName}
            onChangeText={setTeamName}
            style={{ borderWidth: 1, borderColor: "#475569", borderRadius: 8, color: "#fff", padding: 10, marginBottom: 8 }}
          />
          <Button label="Create Team" variant="primary" onPress={createTeam} />
        </View>
      ) : null}

      {["manage", "approvals", "links"].includes(tab) ? (
        <View style={{ marginTop: 8, marginBottom: 8 }}>
          <Text style={{ color: "#e2e8f0", marginBottom: 6 }}>Select Team</Text>
          <View style={{ flexDirection: "row", flexWrap: "wrap", gap: 8 }}>
            {teams.map((team) => {
              const id = getTeamId(team);
              const selected = selectedTeamId === id;
              return (
                <TouchableOpacity
                  key={`select-team-${id}`}
                  onPress={() => setSelectedTeamId(id)}
                  style={{
                    paddingVertical: 6,
                    paddingHorizontal: 10,
                    borderRadius: 999,
                    backgroundColor: selected ? "#f97316" : "#334155"
                  }}
                >
                  <Text style={{ color: "white" }}>{team.TeamName || team.teamName || "Team"}</Text>
                </TouchableOpacity>
              );
            })}
          </View>
          {!teams.length ? <Text style={{ color: "#94a3b8", marginTop: 6 }}>Create a team first.</Text> : null}
        </View>
      ) : null}

      {tab === "manage" ? (
        !selectedTeamId || !teamDetail ? (
          <Text style={{ color: "#94a3b8" }}>Select a team to manage.</Text>
        ) : (
          <>
            <SectionCard title="Update Team">
              <TextInput
                placeholder="Team name"
                placeholderTextColor="#94a3b8"
                value={teamEditName}
                onChangeText={setTeamEditName}
                style={{ borderWidth: 1, borderColor: "#475569", borderRadius: 8, color: "#fff", padding: 10, marginBottom: 8 }}
              />
              <TabRow
                current={teamEditStatus}
                onChange={setTeamEditStatus}
                items={[
                  { value: "active", label: "Active" },
                  { value: "inactive", label: "Inactive" }
                ]}
              />
              <Button label="Save Team" variant="primary" onPress={updateTeamDetails} />
              <Button label="Delete Team" variant="danger" onPress={deleteTeam} />
            </SectionCard>

            <SectionCard title="Add Player">
              <TextInput
                placeholder="Player username or email"
                placeholderTextColor="#94a3b8"
                value={playerIdentifier}
                onChangeText={setPlayerIdentifier}
                style={{ borderWidth: 1, borderColor: "#475569", borderRadius: 8, color: "#fff", padding: 10, marginBottom: 8 }}
              />
              <Button label="Add Player" variant="primary" onPress={addPlayerToTeam} />
            </SectionCard>

            <SectionCard title={`Players (${safeArray(teamDetail.players).length})`}>
              {!safeArray(teamDetail.players).length ? <Text style={{ color: "#94a3b8" }}>No players yet.</Text> : null}
              {safeArray(teamDetail.players).map((player) => (
                <ListItem key={player.id || player.playerId || player.userId} title={player.fullName || player.username || "Player"}>
                  <Text style={{ color: "#cbd5e1" }}>Username: {player.username || "-"}</Text>
                  <Button label="Remove" variant="danger" onPress={() => removePlayerFromTeam(player.id || player.playerId || player.userId)} />
                </ListItem>
              ))}
            </SectionCard>
          </>
        )
      ) : null}

      {tab === "approvals" ? (
        !selectedTeamId ? (
          <Text style={{ color: "#94a3b8" }}>Select a team to view approvals.</Text>
        ) : (
          <>
            {!pendingApprovals.length ? <Text style={{ color: "#94a3b8" }}>No pending approvals.</Text> : null}
            {pendingApprovals.map((approval) => (
              <ListItem key={approval.ID || approval._id} title={approval.AcceptorName || approval.acceptorName || "Pending Request"}>
                <Text style={{ color: "#cbd5e1" }}>Username: {approval.AcceptorUsername || approval.acceptorUsername || "-"}</Text>
                <Text style={{ color: "#cbd5e1" }}>Role: {approval.AcceptorRole || approval.acceptorRole || "-"}</Text>
                <View style={{ marginTop: 8 }}>
                  <Button label="Approve" variant="primary" onPress={() => handleOwnerApproval(approval.ID || approval._id, "approve")} />
                  <Button label="Reject" variant="danger" onPress={() => handleOwnerApproval(approval.ID || approval._id, "reject")} />
                </View>
              </ListItem>
            ))}
          </>
        )
      ) : null}

      {tab === "links" ? (
        !selectedTeamId ? (
          <Text style={{ color: "#94a3b8" }}>Select a team to manage invite links.</Text>
        ) : (
          <>
            <SectionCard title="Create Invite Link">
              <TabRow
                current={linkExpiresIn}
                onChange={setLinkExpiresIn}
                items={[
                  { value: "1h", label: "1h" },
                  { value: "24h", label: "24h" },
                  { value: "7d", label: "7d" },
                  { value: "30d", label: "30d" },
                  { value: "never", label: "Never" }
                ]}
              />
              <TextInput
                placeholder="Max uses (optional)"
                placeholderTextColor="#94a3b8"
                keyboardType="numeric"
                value={linkMaxUses}
                onChangeText={setLinkMaxUses}
                style={{ borderWidth: 1, borderColor: "#475569", borderRadius: 8, color: "#fff", padding: 10, marginBottom: 8 }}
              />
              <Button label="Create Link" variant="primary" onPress={createOwnerInviteLink} />
            </SectionCard>

            {!inviteLinks.length ? <Text style={{ color: "#94a3b8" }}>No invite links for this team.</Text> : null}
            {inviteLinks.map((link) => (
              <ListItem key={link.id} title={link.targetName || "Team Link"}>
                <Text style={{ color: "#cbd5e1" }}>Code: {link.code}</Text>
                <Text style={{ color: "#cbd5e1" }}>Uses: {link.usesCount || 0}{link.maxUses ? `/${link.maxUses}` : ""}</Text>
                <Button label="Deactivate" variant="danger" onPress={() => deleteOwnerInviteLink(link.id)} />
              </ListItem>
            ))}
          </>
        )
      ) : null}

      {tab === "advanced" ? (
        <>
          <SectionCard title="Create Team Invite (/api/teams/:id/invite)">
            <Text style={{ color: "#e2e8f0", marginBottom: 6 }}>Select Team</Text>
            <View style={{ flexDirection: "row", flexWrap: "wrap", gap: 8, marginBottom: 8 }}>
              {teams.map((team) => {
                const id = getTeamId(team);
                const selected = inviteTeamId === id;
                return (
                  <TouchableOpacity key={`invite-team-${id}`} onPress={() => setInviteTeamId(id)} style={{ paddingVertical: 6, paddingHorizontal: 10, borderRadius: 999, backgroundColor: selected ? "#f97316" : "#334155" }}>
                    <Text style={{ color: "white" }}>{team.TeamName || team.teamName || "Team"}</Text>
                  </TouchableOpacity>
                );
              })}
            </View>
            <TextInput
              placeholder="Player username/email"
              placeholderTextColor="#94a3b8"
              value={invitePlayerIdentifier}
              onChangeText={setInvitePlayerIdentifier}
              style={{ borderWidth: 1, borderColor: "#475569", borderRadius: 8, color: "#fff", padding: 10, marginBottom: 8 }}
            />
            <Button label="Send Team Invite" variant="primary" onPress={createTeamInvite} />
            <Button label="Generate Legacy Team Link" onPress={generateLegacyTeamLink} />
            <Button label="Load Team Invites" onPress={() => loadTeamInvites()} />
            {generatedTeamLink ? <Text style={{ color: "#cbd5e1", marginBottom: 6 }}>Link: {generatedTeamLink}</Text> : null}
            {!teamInvites.length ? <Text style={{ color: "#94a3b8" }}>No team invites loaded.</Text> : null}
            {teamInvites.map((invite) => (
              <Text key={getInviteId(invite)} style={{ color: "#cbd5e1", marginBottom: 4 }}>
                {invite.toName || invite.playerName || invite.toUsername || "Invite"} - {normalizeStatus(invite)}
              </Text>
            ))}
          </SectionCard>

          <SectionCard title="Tournament Requests (/api/owner/tournament-requests)">
            <Button label="Load Tournament Requests" onPress={loadOwnerTournamentRequests} />
            {!tournamentRequests.length ? <Text style={{ color: "#94a3b8" }}>No tournament requests loaded.</Text> : null}
            {tournamentRequests.map((req, index) => (
              <Text key={req.id || req._id || index} style={{ color: "#cbd5e1", marginBottom: 4 }}>
                {req.eventName || req.tournamentName || "Request"} - {req.status || "pending"}
              </Text>
            ))}
          </SectionCard>

          <SectionCard title="Team Alias Lookup (/api/team/:id)">
            <TextInput
              placeholder="Team ID"
              placeholderTextColor="#94a3b8"
              value={lookupTeamId}
              onChangeText={setLookupTeamId}
              style={{ borderWidth: 1, borderColor: "#475569", borderRadius: 8, color: "#fff", padding: 10, marginBottom: 8 }}
            />
            <Button label="Lookup Team" onPress={lookupTeamAlias} />
            {lookupTeamData ? <Text style={{ color: "#cbd5e1" }}>{JSON.stringify(lookupTeamData)}</Text> : null}
          </SectionCard>

          <SectionCard title="Matches + Endgame + Raid">
            <TextInput
              placeholder="Match ID for /api/matches/:id"
              placeholderTextColor="#94a3b8"
              value={matchLookupId}
              onChangeText={setMatchLookupId}
              style={{ borderWidth: 1, borderColor: "#475569", borderRadius: 8, color: "#fff", padding: 10, marginBottom: 8 }}
            />
            <Button label="Open /api/matches" onPress={openMatchesRoute} />
            <Button label="Open /api/matches/:id" onPress={openMatchRoute} />
            <Button label="Call /api/endgame" onPress={callEndgameApi} />

            <TabRow
              current={raidType}
              onChange={setRaidType}
              items={[
                { value: "successful", label: "successful" },
                { value: "defense", label: "defense" },
                { value: "empty", label: "empty" }
              ]}
            />
            <TextInput
              placeholder="Raider ID"
              placeholderTextColor="#94a3b8"
              value={raiderId}
              onChangeText={setRaiderId}
              style={{ borderWidth: 1, borderColor: "#475569", borderRadius: 8, color: "#fff", padding: 10, marginBottom: 8 }}
            />
            <TextInput
              placeholder="Defender IDs (comma separated)"
              placeholderTextColor="#94a3b8"
              value={defenderIds}
              onChangeText={setDefenderIds}
              style={{ borderWidth: 1, borderColor: "#475569", borderRadius: 8, color: "#fff", padding: 10, marginBottom: 8 }}
            />
            <Button label={bonusTaken ? "Bonus: ON" : "Bonus: OFF"} onPress={() => setBonusTaken((value) => !value)} />
            <Button label="Submit Raid (/api/matches/raid)" variant="primary" onPress={submitRaidResult} />
          </SectionCard>
        </>
      ) : null}
    </SectionCard>
  );
}

function OrganizerPanel() {
  const [tab, setTab] = useState("events");
  const [eventFilter, setEventFilter] = useState("ongoing");
  const [inviteFilter, setInviteFilter] = useState("pending");
  const [events, setEvents] = useState([]);
  const [invites, setInvites] = useState({ pending: [], accepted: [], declined: [] });
  const [createModel, setCreateModel] = useState({ event_name: "", event_type: "match", max_teams: "4" });
  const [selectedEventId, setSelectedEventId] = useState("");
  const [eventDetail, setEventDetail] = useState(null);
  const [eventEditName, setEventEditName] = useState("");
  const [eventEditType, setEventEditType] = useState("match");
  const [eventEditMaxTeams, setEventEditMaxTeams] = useState("4");
  const [pendingApprovals, setPendingApprovals] = useState([]);
  const [inviteLinks, setInviteLinks] = useState([]);
  const [eventTeams, setEventTeams] = useState([]);
  const [eventMatchStats, setEventMatchStats] = useState(null);
  const [tournamentFixtures, setTournamentFixtures] = useState([]);
  const [tournamentStandings, setTournamentStandings] = useState([]);
  const [championshipInfo, setChampionshipInfo] = useState(null);
  const [championshipFixtures, setChampionshipFixtures] = useState([]);
  const [championshipStats, setChampionshipStats] = useState([]);
  const [matchLookupId, setMatchLookupId] = useState("");
  const [playerSelectionMatchId, setPlayerSelectionMatchId] = useState("");
  const [raidType, setRaidType] = useState("successful");
  const [raiderId, setRaiderId] = useState("");
  const [defenderIds, setDefenderIds] = useState("");
  const [bonusTaken, setBonusTaken] = useState(false);
  const [generatedEventLink, setGeneratedEventLink] = useState("");
  const [ownerIdentifier, setOwnerIdentifier] = useState("");
  const [linkExpiresIn, setLinkExpiresIn] = useState("30d");
  const [linkMaxUses, setLinkMaxUses] = useState("");
  const [message, setMessage] = useState("");
  const [messageType, setMessageType] = useState("info");

  async function loadData() {
    setMessage("");
    try {
      const [eventsRes, invitesRes] = await Promise.all([
        authClient.apiFetch("/api/organizer/events"),
        authClient.apiFetch("/api/organizer/event-invites")
      ]);
      const eventData = safeArray(await eventsRes.json());
      const inviteData = safeArray(await invitesRes.json());
      setEvents(eventData);
      setInvites({
        pending: inviteData.filter((invite) => normalizeStatus(invite) === "pending"),
        accepted: inviteData.filter((invite) => normalizeStatus(invite) === "accepted"),
        declined: inviteData.filter((invite) => normalizeStatus(invite) === "declined")
      });
      if (!selectedEventId && eventData.length > 0) {
        setSelectedEventId(eventData[0].id || "");
      }
    } catch (error) {
      setMessageType("danger");
      setMessage(error.message || "Failed to load organizer data");
    }
  }

  async function loadEventContext(eventId) {
    if (!eventId) {
      setEventDetail(null);
      setPendingApprovals([]);
      setInviteLinks([]);
      return;
    }

    try {
      const [eventRes, approvalsRes, linksRes] = await Promise.all([
        authClient.apiFetch(`/api/organizer/events/${eventId}`),
        authClient.apiFetch(`/api/events/${eventId}/pending-approvals`),
        authClient.apiFetch("/api/organizer/invite-links")
      ]);
      const eventData = await eventRes.json().catch(() => ({}));
      const approvalsData = await approvalsRes.json().catch(() => []);
      const linksData = await linksRes.json().catch(() => ({}));

      if (!eventRes.ok) throw new Error(eventData.error || "Failed to load event details");
      if (!approvalsRes.ok) throw new Error(approvalsData.error || "Failed to load pending approvals");
      if (!linksRes.ok) throw new Error(linksData.error || "Failed to load invite links");

      setEventDetail(eventData);
      setEventEditName(eventData.eventName || "");
      setEventEditType(eventData.eventType || "match");
      setEventEditMaxTeams(String(eventData.maxTeams || 4));
      setPendingApprovals(safeArray(approvalsData));
      setInviteLinks(safeArray(linksData.data).filter((link) => (link.targetId || "") === eventId));
      await loadEventOpsData(eventId, eventData.eventType || "match");
    } catch (error) {
      setMessageType("danger");
      setMessage(error.message || "Failed to load event context");
    }
  }

  async function loadEventOpsData(eventId, eventType) {
    if (!eventId) {
      setEventTeams([]);
      setEventMatchStats(null);
      setTournamentFixtures([]);
      setTournamentStandings([]);
      setChampionshipInfo(null);
      setChampionshipFixtures([]);
      setChampionshipStats([]);
      return;
    }

    try {
      const [teamsRes, matchStatsRes] = await Promise.all([
        authClient.apiFetch(`/api/events/${eventId}/teams`),
        authClient.apiFetch(`/api/organizer/events/${eventId}/match`)
      ]);
      const teamsData = await teamsRes.json().catch(() => []);
      const matchStatsData = await matchStatsRes.json().catch(() => ({}));

      if (teamsRes.ok) setEventTeams(safeArray(teamsData));
      if (matchStatsRes.ok) {
        setEventMatchStats(matchStatsData);
      } else {
        setEventMatchStats(null);
      }

      if (eventType === "tournament") {
        const [fixturesRes, standingsRes] = await Promise.all([
          authClient.apiFetch(`/api/tournaments/${eventId}/fixtures`),
          authClient.apiFetch(`/api/tournaments/${eventId}/standings`)
        ]);
        const fixturesData = await fixturesRes.json().catch(() => ({}));
        const standingsData = await standingsRes.json().catch(() => ({}));
        if (fixturesRes.ok) setTournamentFixtures(safeArray(fixturesData.fixtures));
        if (standingsRes.ok) setTournamentStandings(safeArray(standingsData.standings));
      } else {
        setTournamentFixtures([]);
        setTournamentStandings([]);
      }

      if (eventType === "championship") {
        const [champRes, champFixturesRes, champStatsRes] = await Promise.all([
          authClient.apiFetch(`/api/championships/${eventId}`),
          authClient.apiFetch(`/api/championships/${eventId}/fixtures`),
          authClient.apiFetch(`/api/championships/${eventId}/stats`)
        ]);
        const champData = await champRes.json().catch(() => ({}));
        const champFixturesData = await champFixturesRes.json().catch(() => []);
        const champStatsData = await champStatsRes.json().catch(() => []);
        if (champRes.ok) setChampionshipInfo(champData);
        if (champFixturesRes.ok) setChampionshipFixtures(safeArray(champFixturesData));
        if (champStatsRes.ok) setChampionshipStats(safeArray(champStatsData));
      } else {
        setChampionshipInfo(null);
        setChampionshipFixtures([]);
        setChampionshipStats([]);
      }
    } catch {
      setEventTeams([]);
      setEventMatchStats(null);
      setTournamentFixtures([]);
      setTournamentStandings([]);
      setChampionshipInfo(null);
      setChampionshipFixtures([]);
      setChampionshipStats([]);
    }
  }

  useEffect(() => {
    loadData();
  }, []);

  useEffect(() => {
    if (!selectedEventId) return;
    loadEventContext(selectedEventId);
  }, [selectedEventId]);

  const filteredEvents = useMemo(() => {
    return events.filter((event) => {
      const status = String(event.status || "").toLowerCase();
      if (eventFilter === "ongoing") return status === "active" || status === "ongoing";
      if (eventFilter === "pending") return status !== "completed" && status !== "active" && status !== "ongoing";
      if (eventFilter === "completed") return status === "completed";
      return true;
    });
  }, [events, eventFilter]);

  async function createEvent() {
    if (!createModel.event_name.trim()) {
      setMessageType("warning");
      setMessage("Event name is required");
      return;
    }
    if (["tournament", "championship"].includes(createModel.event_type) && Number(createModel.max_teams) < 4) {
      setMessageType("warning");
      setMessage("Number of teams must be at least 4 for tournament/championship");
      return;
    }
    try {
      const response = await authClient.apiFetch("/api/events", {
        method: "POST",
        body: JSON.stringify({
          event_name: createModel.event_name,
          event_type: createModel.event_type,
          max_teams: Number(createModel.max_teams || 0)
        })
      });
      const data = await response.json().catch(() => ({}));
      if (!response.ok) throw new Error(data.error || "Failed to create event");
      setCreateModel({ event_name: "", event_type: "match", max_teams: "4" });
      setTab("events");
      setMessageType("success");
      setMessage(`Event created: ${data.event_name || "new event"}`);
      await loadData();
    } catch (error) {
      setMessageType("danger");
      setMessage(error.message || "Failed to create event");
    }
  }

  async function updateEventDetails() {
    if (!selectedEventId) return;
    try {
      const response = await authClient.apiFetch(`/api/events/${selectedEventId}`, {
        method: "PUT",
        body: JSON.stringify({
          event_name: eventEditName,
          event_type: eventEditType,
          max_teams: Number(eventEditMaxTeams || 0)
        })
      });
      const data = await response.json().catch(() => ({}));
      if (!response.ok) throw new Error(data.error || "Failed to update event");
      setMessageType("success");
      setMessage("Event updated");
      await loadData();
      await loadEventContext(selectedEventId);
    } catch (error) {
      setMessageType("danger");
      setMessage(error.message || "Failed to update event");
    }
  }

  async function completeEvent() {
    if (!selectedEventId) return;
    try {
      const response = await authClient.apiFetch(`/api/events/${selectedEventId}/complete`, { method: "POST" });
      const data = await response.json().catch(() => ({}));
      if (!response.ok) throw new Error(data.error || "Failed to complete event");
      setMessageType("success");
      setMessage("Event marked completed");
      await loadData();
      await loadEventContext(selectedEventId);
    } catch (error) {
      setMessageType("danger");
      setMessage(error.message || "Failed to complete event");
    }
  }

  async function startEvent() {
    if (!selectedEventId) return;
    try {
      const response = await authClient.apiFetch(`/api/organizer/events/${selectedEventId}/start`, { method: "POST" });
      const data = await response.json().catch(() => ({}));
      if (!response.ok) throw new Error(data.error || "Failed to start event");
      setMessageType("success");
      setMessage("Event started");
      await loadData();
      await loadEventContext(selectedEventId);
    } catch (error) {
      setMessageType("danger");
      setMessage(error.message || "Failed to start event");
    }
  }

  async function sendInvite() {
    if (!selectedEventId || !ownerIdentifier.trim()) return;
    try {
      const response = await authClient.apiFetch(`/api/events/${selectedEventId}/invite`, {
        method: "POST",
        body: JSON.stringify({ ownerIdentifier: ownerIdentifier.trim() })
      });
      const data = await response.json().catch(() => ({}));
      if (!response.ok) throw new Error(data.error || "Failed to send invite");
      setOwnerIdentifier("");
      setMessageType("success");
      setMessage("Invitation created");
      await loadData();
      await loadEventContext(selectedEventId);
    } catch (error) {
      setMessageType("danger");
      setMessage(error.message || "Failed to send invite");
    }
  }

  async function handleOrganizerApproval(approvalId, action) {
    const route = action === "approve"
      ? fillRoute(ROUTE_TEMPLATES.pendingApprovalApprove, { id: approvalId })
      : fillRoute(ROUTE_TEMPLATES.pendingApprovalReject, { id: approvalId });
    try {
      const response = await authClient.apiFetch(route, { method: "PUT" });
      const data = await response.json().catch(() => ({}));
      if (!response.ok) throw new Error(data.error || `Failed to ${action} approval`);
      setMessageType("success");
      setMessage(data.message || `Approval ${action}d`);
      await loadEventContext(selectedEventId);
      await loadData();
    } catch (error) {
      setMessageType("danger");
      setMessage(error.message || `Failed to ${action} approval`);
    }
  }

  async function createOrganizerInviteLink() {
    if (!selectedEventId) return;
    const payload = {
      targetId: selectedEventId,
      expiresIn: linkExpiresIn
    };
    if (Number(linkMaxUses) > 0) payload.maxUses = Number(linkMaxUses);
    try {
      const response = await authClient.apiFetch("/api/organizer/invite-links", {
        method: "POST",
        body: JSON.stringify(payload)
      });
      const data = await response.json().catch(() => ({}));
      if (!response.ok) throw new Error(data.error || "Failed to create invite link");
      setLinkMaxUses("");
      setMessageType("success");
      setMessage("Invite link created");
      await loadEventContext(selectedEventId);
    } catch (error) {
      setMessageType("danger");
      setMessage(error.message || "Failed to create invite link");
    }
  }

  async function deleteOrganizerInviteLink(linkId) {

      async function openMatchesRoute() {
        try {
          const response = await authClient.apiFetch("/api/matches");
          const html = await response.text();
          if (!response.ok) throw new Error("Failed to open /api/matches route");
          setMessageType("info");
          setMessage(`Matches route opened, HTML length: ${html.length}`);
        } catch (error) {
          setMessageType("danger");
          setMessage(error.message || "Failed to open matches route");
        }
      }

      async function openMatchRoute() {
        if (!matchLookupId.trim()) return;
        try {
          const response = await authClient.apiFetch(`/api/matches/${matchLookupId.trim()}`);
          const html = await response.text();
          if (!response.ok) throw new Error("Failed to open /api/matches/:id route");
          setMessageType("info");
          setMessage(`Match route opened, HTML length: ${html.length}`);
        } catch (error) {
          setMessageType("danger");
          setMessage(error.message || "Failed to open match route");
        }
      }

      async function callEndgameApi() {
        try {
          const response = await authClient.apiFetch("/api/endgame");
          const body = await response.text();
          if (!response.ok) throw new Error("Failed to call /api/endgame");
          setMessageType("info");
          setMessage(`/api/endgame response length: ${body.length}`);
        } catch (error) {
          setMessageType("danger");
          setMessage(error.message || "Failed to call /api/endgame");
        }
      }

      async function checkPlayerSelectionRoute() {
        if (!playerSelectionMatchId.trim()) return;
        try {
          const response = await authClient.apiFetch(`/organizer/playerselection/${playerSelectionMatchId.trim()}`);
          const html = await response.text();
          if (!response.ok) throw new Error("Failed to open /organizer/playerselection/:id route");
          setMessageType("info");
          setMessage(`Player selection route opened, HTML length: ${html.length}`);
        } catch (error) {
          setMessageType("danger");
          setMessage(error.message || "Failed to open player selection route");
        }
      }

      async function checkScorerRoute() {
        try {
          const response = await authClient.apiFetch("/scorer");
          const html = await response.text();
          if (!response.ok) throw new Error("Failed to open /scorer route");
          setMessageType("info");
          setMessage(`Scorer route opened, HTML length: ${html.length}`);
        } catch (error) {
          setMessageType("danger");
          setMessage(error.message || "Failed to open scorer route");
        }
      }

      async function submitRaidResult() {
        if (!raiderId.trim()) return;
        const payload = {
          raidType,
          raiderId: raiderId.trim(),
          defenderIds: defenderIds.split(",").map((item) => item.trim()).filter(Boolean),
          bonusTaken
        };
        try {
          const response = await authClient.apiFetch("/api/matches/raid", {
            method: "POST",
            body: JSON.stringify(payload)
          });
          const data = await response.json().catch(() => ({}));
          if (!response.ok) throw new Error(data.error || "Failed to process raid");
          setMessageType("success");
          setMessage("Raid processed successfully");
        } catch (error) {
          setMessageType("danger");
          setMessage(error.message || "Failed to process raid");
        }
      }
    try {
      const response = await authClient.apiFetch(`/api/organizer/invite-links/${linkId}`, { method: "DELETE" });
      const data = await response.json().catch(() => ({}));
      if (!response.ok) throw new Error(data.error || "Failed to delete invite link");
      setMessageType("success");
      setMessage("Invite link deleted");
      await loadEventContext(selectedEventId);
    } catch (error) {
      setMessageType("danger");
      setMessage(error.message || "Failed to delete invite link");
    }
  }

  async function generateLegacyEventLink() {
    if (!selectedEventId) return;
    try {
      const response = await authClient.apiFetch(`/api/events/${selectedEventId}/generate-link`, { method: "POST" });
      const data = await response.json().catch(() => ({}));
      if (!response.ok) throw new Error(data.error || "Failed to generate event invite link");
      setGeneratedEventLink(data.invite_url || (data.invite_token ? `/invite/event/${data.invite_token}` : ""));
      setMessageType("success");
      setMessage("Legacy event invite link generated");
    } catch (error) {
      setMessageType("danger");
      setMessage(error.message || "Failed to generate event invite link");
    }
  }

  async function initializeTournament() {
    if (!selectedEventId) return;
    try {
      const response = await authClient.apiFetch(`/api/tournaments/initialize/${selectedEventId}`, { method: "POST" });
      const data = await response.json().catch(() => ({}));
      if (!response.ok) throw new Error(data.error || "Failed to initialize tournament");
      setMessageType("success");
      setMessage(data.message || "Tournament initialized");
      await loadEventContext(selectedEventId);
    } catch (error) {
      setMessageType("danger");
      setMessage(error.message || "Failed to initialize tournament");
    }
  }

  async function startTournamentFixture(fixtureId) {
    if (!selectedEventId || !fixtureId) return;
    try {
      const response = await authClient.apiFetch(`/api/tournaments/${selectedEventId}/start-match/${fixtureId}`, { method: "POST" });
      const data = await response.json().catch(() => ({}));
      if (!response.ok) throw new Error(data.error || "Failed to start fixture match");
      setMessageType("success");
      setMessage(`Tournament match started: ${data.matchId || "created"}`);
      await loadEventContext(selectedEventId);
    } catch (error) {
      setMessageType("danger");
      setMessage(error.message || "Failed to start fixture match");
    }
  }

  async function initializeChampionship() {
    if (!selectedEventId) return;
    try {
      const response = await authClient.apiFetch(`/api/championships/initialize/${selectedEventId}`, { method: "POST" });
      const data = await response.json().catch(() => ({}));
      if (!response.ok) throw new Error(data.error || "Failed to initialize championship");
      setMessageType("success");
      setMessage(data.message || "Championship initialized");
      await loadEventContext(selectedEventId);
    } catch (error) {
      setMessageType("danger");
      setMessage(error.message || "Failed to initialize championship");
    }
  }

  async function startChampionshipFixture(fixtureId) {
    if (!fixtureId) return;
    const championshipId = getObjectIdString(championshipInfo?.id || championshipInfo?._id);
    if (!championshipId) {
      setMessageType("warning");
      setMessage("Championship ID not found");
      return;
    }

    try {
      const response = await authClient.apiFetch(`/api/championships/${championshipId}/start-match/${fixtureId}`, { method: "POST" });
      const data = await response.json().catch(() => ({}));
      if (!response.ok) throw new Error(data.error || "Failed to start championship match");
      setMessageType("success");
      setMessage(`Championship match started: ${data.matchId || "created"}`);
      await loadEventContext(selectedEventId);
    } catch (error) {
      setMessageType("danger");
      setMessage(error.message || "Failed to start championship match");
    }
  }

  return (
    <SectionCard title="Organizer" subtitle="Manage your events and invite responses.">
      <StatusBox message={message} type={messageType} />
      <Button label="Refresh" variant="primary" onPress={loadData} />
      <TabRow
        current={tab}
        onChange={setTab}
        items={[
          { value: "events", label: "My Events" },
          { value: "invites", label: "Invite Responses" },
          { value: "manage", label: "Event Manager" },
          { value: "competition", label: "Tournament/Championship" },
          { value: "advanced", label: "Advanced APIs" },
          { value: "approvals", label: `Pending Approvals (${pendingApprovals.length})` },
          { value: "links", label: `Invite Links (${inviteLinks.length})` },
          { value: "create", label: "Create Event" }
        ]}
      />

      {tab === "events" ? (
        <>
          <TabRow
            current={eventFilter}
            onChange={setEventFilter}
            items={[
              { value: "ongoing", label: "Ongoing" },
              { value: "pending", label: "Pending" },
              { value: "completed", label: "Completed" }
            ]}
          />
          {!filteredEvents.length ? <Text style={{ color: "#94a3b8" }}>No events in this state.</Text> : null}
          {filteredEvents.map((event) => (
            <ListItem key={event.id} title={event.eventName || "Event"}>
              <Text style={{ color: "#cbd5e1" }}>Type: {event.eventType}</Text>
              <Text style={{ color: "#cbd5e1" }}>Status: {event.status}</Text>
              <Text style={{ color: "#cbd5e1" }}>Accepted: {event.counts?.accepted || 0} | Pending: {event.counts?.pending || 0}</Text>
            </ListItem>
          ))}
        </>
      ) : null}

      {tab === "invites" ? (
        <>
          <TabRow
            current={inviteFilter}
            onChange={setInviteFilter}
            items={[
              { value: "pending", label: `Pending (${invites.pending.length})` },
              { value: "accepted", label: `Accepted (${invites.accepted.length})` },
              { value: "declined", label: `Declined (${invites.declined.length})` }
            ]}
          />
          {!(invites[inviteFilter] || []).length ? <Text style={{ color: "#94a3b8" }}>No invite responses in this tab.</Text> : null}
          {(invites[inviteFilter] || []).map((invite) => (
            <ListItem key={getInviteId(invite)} title={invite.eventName || "Event Invite"}>
              <Text style={{ color: "#cbd5e1" }}>Owner: {invite.ownerName || "Unknown"}</Text>
              <Text style={{ color: "#cbd5e1" }}>Team: {invite.teamName || "Unassigned"}</Text>
              <Text style={{ color: "#cbd5e1" }}>Status: {getStatusDisplay(normalizeStatus(invite))}</Text>
              {getDeclineReason(invite) ? <Text style={{ color: "#cbd5e1" }}>Reason: {getDeclineReason(invite)}</Text> : null}
            </ListItem>
          ))}
        </>
      ) : null}

      {tab === "create" ? (
        <View>
          <TextInput
            placeholder="Event name"
            placeholderTextColor="#94a3b8"
            value={createModel.event_name}
            onChangeText={(value) => setCreateModel((prev) => ({ ...prev, event_name: value }))}
            style={{ borderWidth: 1, borderColor: "#475569", borderRadius: 8, color: "#fff", padding: 10, marginBottom: 8 }}
          />
          <TabRow
            current={createModel.event_type}
            onChange={(value) => setCreateModel((prev) => ({ ...prev, event_type: value }))}
            items={[
              { value: "match", label: "Match" },
              { value: "tournament", label: "Tournament" },
              { value: "championship", label: "Championship" }
            ]}
          />
          {createModel.event_type !== "match" ? (
            <TextInput
              placeholder="No. of Teams (>= 4)"
              placeholderTextColor="#94a3b8"
              keyboardType="numeric"
              value={String(createModel.max_teams)}
              onChangeText={(value) => setCreateModel((prev) => ({ ...prev, max_teams: value }))}
              style={{ borderWidth: 1, borderColor: "#475569", borderRadius: 8, color: "#fff", padding: 10, marginBottom: 8 }}
            />
          ) : null}
          <Button label="Create Event" variant="primary" onPress={createEvent} />
        </View>
      ) : null}

      {["manage", "competition", "advanced", "approvals", "links"].includes(tab) ? (
        <View style={{ marginTop: 8, marginBottom: 8 }}>
          <Text style={{ color: "#e2e8f0", marginBottom: 6 }}>Select Event</Text>
          <View style={{ flexDirection: "row", flexWrap: "wrap", gap: 8 }}>
            {events.map((event) => {
              const id = event.id || "";
              const selected = selectedEventId === id;
              return (
                <TouchableOpacity
                  key={`select-event-${id}`}
                  onPress={() => setSelectedEventId(id)}
                  style={{
                    paddingVertical: 6,
                    paddingHorizontal: 10,
                    borderRadius: 999,
                    backgroundColor: selected ? "#f97316" : "#334155"
                  }}
                >
                  <Text style={{ color: "white" }}>{event.eventName || "Event"}</Text>
                </TouchableOpacity>
              );
            })}
          </View>
          {!events.length ? <Text style={{ color: "#94a3b8", marginTop: 6 }}>Create an event first.</Text> : null}
        </View>
      ) : null}

      {tab === "manage" ? (
        !selectedEventId || !eventDetail ? (
          <Text style={{ color: "#94a3b8" }}>Select an event to manage.</Text>
        ) : (
          <>
            <SectionCard title="Update Event">
              <TextInput
                placeholder="Event name"
                placeholderTextColor="#94a3b8"
                value={eventEditName}
                onChangeText={setEventEditName}
                style={{ borderWidth: 1, borderColor: "#475569", borderRadius: 8, color: "#fff", padding: 10, marginBottom: 8 }}
              />
              <TabRow
                current={eventEditType}
                onChange={setEventEditType}
                items={[
                  { value: "match", label: "Match" },
                  { value: "tournament", label: "Tournament" },
                  { value: "championship", label: "Championship" }
                ]}
              />
              {eventEditType !== "match" ? (
                <TextInput
                  placeholder="No. of teams"
                  placeholderTextColor="#94a3b8"
                  keyboardType="numeric"
                  value={String(eventEditMaxTeams)}
                  onChangeText={setEventEditMaxTeams}
                  style={{ borderWidth: 1, borderColor: "#475569", borderRadius: 8, color: "#fff", padding: 10, marginBottom: 8 }}
                />
              ) : null}
              <Button label="Save Event" variant="primary" onPress={updateEventDetails} />
              <Button label="Start Event" variant="primary" onPress={startEvent} />
              <Button label="Mark Completed" onPress={completeEvent} />
            </SectionCard>

            <SectionCard title="Direct Invite">
              <TextInput
                placeholder="Owner username or email"
                placeholderTextColor="#94a3b8"
                value={ownerIdentifier}
                onChangeText={setOwnerIdentifier}
                style={{ borderWidth: 1, borderColor: "#475569", borderRadius: 8, color: "#fff", padding: 10, marginBottom: 8 }}
              />
              <Button label="Send Invite" variant="primary" onPress={sendInvite} />
            </SectionCard>

            <SectionCard title={`Participating Teams (${eventTeams.length})`}>
              {!eventTeams.length ? <Text style={{ color: "#94a3b8" }}>No accepted teams yet.</Text> : null}
              {eventTeams.map((team) => (
                <Text key={team.teamId || team.TeamID || team.ID || team.teamName} style={{ color: "#cbd5e1", marginBottom: 4 }}>
                  {(team.teamName || team.TeamName || team.name || "Team")} - {(team.status || team.Status || "accepted")}
                </Text>
              ))}
            </SectionCard>

            <SectionCard title="Latest Match Stats">
              {!eventMatchStats ? <Text style={{ color: "#94a3b8" }}>No match stats found yet.</Text> : null}
              {eventMatchStats ? (
                <>
                  <Text style={{ color: "#cbd5e1" }}>Match ID: {eventMatchStats.matchId || eventMatchStats._id || "-"}</Text>
                  <Text style={{ color: "#cbd5e1" }}>Team A: {eventMatchStats?.data?.TeamAScore ?? "-"} | Team B: {eventMatchStats?.data?.TeamBScore ?? "-"}</Text>
                </>
              ) : null}
            </SectionCard>
          </>
        )
      ) : null}

      {tab === "competition" ? (
        !selectedEventId || !eventDetail ? (
          <Text style={{ color: "#94a3b8" }}>Select an event to manage competition data.</Text>
        ) : (
          <>
            {eventDetail.eventType === "tournament" ? (
              <>
                <SectionCard title="Tournament Actions">
                  <Button label="Initialize Tournament" variant="primary" onPress={initializeTournament} />
                </SectionCard>

                <SectionCard title={`Fixtures (${tournamentFixtures.length})`}>
                  {!tournamentFixtures.length ? <Text style={{ color: "#94a3b8" }}>No fixtures found.</Text> : null}
                  {tournamentFixtures.map((fixture) => (
                    <ListItem key={fixture.id || fixture._id} title={`${fixture.team1Name || "Team A"} vs ${fixture.team2Name || "Team B"}`}>
                      <Text style={{ color: "#cbd5e1" }}>Status: {fixture.status || "pending"}</Text>
                      {String(fixture.status || "").toLowerCase() === "pending" ? (
                        <Button label="Start Match" onPress={() => startTournamentFixture(fixture.id || fixture._id)} />
                      ) : null}
                    </ListItem>
                  ))}
                </SectionCard>

                <SectionCard title={`Standings (${tournamentStandings.length})`}>
                  {!tournamentStandings.length ? <Text style={{ color: "#94a3b8" }}>No standings yet.</Text> : null}
                  {tournamentStandings.map((item) => (
                    <Text key={`${item.teamId}-${item.position}`} style={{ color: "#cbd5e1", marginBottom: 4 }}>
                      #{item.position} {item.teamName} - {item.points} pts (NRR {item.nrr})
                    </Text>
                  ))}
                </SectionCard>
              </>
            ) : null}

            {eventDetail.eventType === "championship" ? (
              <>
                <SectionCard title="Championship Actions">
                  <Button label="Initialize Championship" variant="primary" onPress={initializeChampionship} />
                </SectionCard>

                <SectionCard title="Championship Details">
                  {!championshipInfo ? <Text style={{ color: "#94a3b8" }}>Not initialized yet.</Text> : null}
                  {championshipInfo ? (
                    <>
                      <Text style={{ color: "#cbd5e1" }}>ID: {getObjectIdString(championshipInfo.id || championshipInfo._id) || "-"}</Text>
                      <Text style={{ color: "#cbd5e1" }}>Status: {championshipInfo.status || "-"}</Text>
                      <Text style={{ color: "#cbd5e1" }}>Round: {championshipInfo.currentRound || "-"}/{championshipInfo.totalRounds || "-"}</Text>
                    </>
                  ) : null}
                </SectionCard>

                <SectionCard title={`Fixtures (${championshipFixtures.length})`}>
                  {!championshipFixtures.length ? <Text style={{ color: "#94a3b8" }}>No fixtures found.</Text> : null}
                  {championshipFixtures.map((fixture) => {
                    const fixtureId = getObjectIdString(fixture.id || fixture._id);
                    return (
                      <ListItem
                        key={fixtureId || `${fixture.roundNumber || "r"}-${getObjectIdString(fixture.team1Id) || "t1"}-${getObjectIdString(fixture.team2Id) || "bye"}`}
                        title={`${fixture.team1?.name || fixture.team1?.teamName || "Team A"} vs ${fixture.team2?.name || fixture.team2?.teamName || (fixture.isBye ? "BYE" : "Team B")}`}
                      >
                        <Text style={{ color: "#cbd5e1" }}>Status: {fixture.status || "pending"}</Text>
                        {String(fixture.status || "").toLowerCase() === "pending" && fixtureId ? (
                          <Button label="Start Match" onPress={() => startChampionshipFixture(fixtureId)} />
                        ) : null}
                      </ListItem>
                    );
                  })}
                </SectionCard>

                <SectionCard title={`Stats (${championshipStats.length})`}>
                  {!championshipStats.length ? <Text style={{ color: "#94a3b8" }}>No stats yet.</Text> : null}
                  {championshipStats.map((item) => (
                    <Text key={getObjectIdString(item.id || item._id) || item.team?.name} style={{ color: "#cbd5e1", marginBottom: 4 }}>
                      {item.team?.name || item.team?.teamName || "Team"} - NRR {item.nrr || 0}
                    </Text>
                  ))}
                </SectionCard>
              </>
            ) : null}
          </>
        )
      ) : null}

      {tab === "advanced" ? (
        <>
          <SectionCard title="Match Routes + Endgame">
            <TextInput
              placeholder="Match ID for /api/matches/:id"
              placeholderTextColor="#94a3b8"
              value={matchLookupId}
              onChangeText={setMatchLookupId}
              style={{ borderWidth: 1, borderColor: "#475569", borderRadius: 8, color: "#fff", padding: 10, marginBottom: 8 }}
            />
            <Button label="Open /api/matches" onPress={openMatchesRoute} />
            <Button label="Open /api/matches/:id" onPress={openMatchRoute} />
            <Button label="Call /api/endgame" onPress={callEndgameApi} />
            <Button label="Generate Legacy Event Link" onPress={generateLegacyEventLink} />
            {generatedEventLink ? <Text style={{ color: "#cbd5e1", marginBottom: 6 }}>Link: {generatedEventLink}</Text> : null}
          </SectionCard>

          <SectionCard title="Legacy Organizer Routes">
            <TextInput
              placeholder="Match ID for /organizer/playerselection/:id"
              placeholderTextColor="#94a3b8"
              value={playerSelectionMatchId}
              onChangeText={setPlayerSelectionMatchId}
              style={{ borderWidth: 1, borderColor: "#475569", borderRadius: 8, color: "#fff", padding: 10, marginBottom: 8 }}
            />
            <Button label="Check /organizer/playerselection/:id" onPress={checkPlayerSelectionRoute} />
            <Button label="Check /scorer" onPress={checkScorerRoute} />
          </SectionCard>

          <SectionCard title="Submit Raid (/api/matches/raid)">
            <TabRow
              current={raidType}
              onChange={setRaidType}
              items={[
                { value: "successful", label: "successful" },
                { value: "defense", label: "defense" },
                { value: "empty", label: "empty" }
              ]}
            />
            <TextInput
              placeholder="Raider ID"
              placeholderTextColor="#94a3b8"
              value={raiderId}
              onChangeText={setRaiderId}
              style={{ borderWidth: 1, borderColor: "#475569", borderRadius: 8, color: "#fff", padding: 10, marginBottom: 8 }}
            />
            <TextInput
              placeholder="Defender IDs (comma separated)"
              placeholderTextColor="#94a3b8"
              value={defenderIds}
              onChangeText={setDefenderIds}
              style={{ borderWidth: 1, borderColor: "#475569", borderRadius: 8, color: "#fff", padding: 10, marginBottom: 8 }}
            />
            <Button label={bonusTaken ? "Bonus: ON" : "Bonus: OFF"} onPress={() => setBonusTaken((value) => !value)} />
            <Button label="Submit Raid" variant="primary" onPress={submitRaidResult} />
          </SectionCard>
        </>
      ) : null}

      {tab === "approvals" ? (
        !selectedEventId ? (
          <Text style={{ color: "#94a3b8" }}>Select an event to view approvals.</Text>
        ) : (
          <>
            {!pendingApprovals.length ? <Text style={{ color: "#94a3b8" }}>No pending approvals.</Text> : null}
            {pendingApprovals.map((approval) => (
              <ListItem key={approval.ID || approval._id} title={approval.AcceptorName || approval.acceptorName || "Pending Request"}>
                <Text style={{ color: "#cbd5e1" }}>Username: {approval.AcceptorUsername || approval.acceptorUsername || "-"}</Text>
                <Text style={{ color: "#cbd5e1" }}>Role: {approval.AcceptorRole || approval.acceptorRole || "-"}</Text>
                <View style={{ marginTop: 8 }}>
                  <Button label="Approve" variant="primary" onPress={() => handleOrganizerApproval(approval.ID || approval._id, "approve")} />
                  <Button label="Reject" variant="danger" onPress={() => handleOrganizerApproval(approval.ID || approval._id, "reject")} />
                </View>
              </ListItem>
            ))}
          </>
        )
      ) : null}

      {tab === "links" ? (
        !selectedEventId ? (
          <Text style={{ color: "#94a3b8" }}>Select an event to manage invite links.</Text>
        ) : (
          <>
            <SectionCard title="Create Invite Link">
              <TabRow
                current={linkExpiresIn}
                onChange={setLinkExpiresIn}
                items={[
                  { value: "1h", label: "1h" },
                  { value: "24h", label: "24h" },
                  { value: "7d", label: "7d" },
                  { value: "30d", label: "30d" },
                  { value: "never", label: "Never" }
                ]}
              />
              <TextInput
                placeholder="Max uses (optional)"
                placeholderTextColor="#94a3b8"
                keyboardType="numeric"
                value={linkMaxUses}
                onChangeText={setLinkMaxUses}
                style={{ borderWidth: 1, borderColor: "#475569", borderRadius: 8, color: "#fff", padding: 10, marginBottom: 8 }}
              />
              <Button label="Create Link" variant="primary" onPress={createOrganizerInviteLink} />
            </SectionCard>

            {!inviteLinks.length ? <Text style={{ color: "#94a3b8" }}>No invite links for this event.</Text> : null}
            {inviteLinks.map((link) => (
              <ListItem key={link.id} title={link.targetName || "Event Link"}>
                <Text style={{ color: "#cbd5e1" }}>Code: {link.code}</Text>
                <Text style={{ color: "#cbd5e1" }}>Uses: {link.usesCount || 0}{link.maxUses ? `/${link.maxUses}` : ""}</Text>
                <Button label="Deactivate" variant="danger" onPress={() => deleteOrganizerInviteLink(link.id)} />
              </ListItem>
            ))}
          </>
        )
      ) : null}
    </SectionCard>
  );
}

export default function App() {
  const [authMode, setAuthMode] = useState("login");
  const [identifier, setIdentifier] = useState("");
  const [password, setPassword] = useState("");
  const [fullName, setFullName] = useState("");
  const [email, setEmail] = useState("");
  const [signupRole, setSignupRole] = useState("player");
  const [position, setPosition] = useState("raider");
  const [confirmPassword, setConfirmPassword] = useState("");
  const [publicTab, setPublicTab] = useState("match");
  const [publicTargetId, setPublicTargetId] = useState("");
  const [inviteToken, setInviteToken] = useState("");
  const [inviteTarget, setInviteTarget] = useState("team");
  const [publicPayload, setPublicPayload] = useState(null);
  const [session, setSession] = useState({ token: null, userId: null, role: null, exp: null });
  const [message, setMessage] = useState("");
  const [messageType, setMessageType] = useState("info");
  const [showProfile, setShowProfile] = useState(false);
  const [profile, setProfile] = useState(null);

  async function reloadSession() {
    const current = await authClient.getSessionSummary();
    setSession(current);
  }

  useEffect(() => {
    reloadSession();
  }, []);

  async function doLogin() {
    setMessage("");
    try {
      const data = await authClient.login({ identifier, password });
      await reloadSession();
      setMessageType("success");
      setMessage(`Logged in as ${normalizeRole(data.role || "player")}`);
    } catch (err) {
      setMessageType("danger");
      setMessage(err.message || "Login failed");
    }
  }

  async function doSignup() {
    setMessage("");
    try {
      await authClient.signup({
        fullName,
        email,
        userId: identifier,
        password,
        confirmPassword,
        role: signupRole,
        position
      });
      setMessageType("success");
      setMessage("Signup successful. Please sign in.");
      setAuthMode("login");
    } catch (err) {
      setMessageType("danger");
      setMessage(err.message || "Signup failed");
    }
  }

  async function loadPublicData() {
    setMessage("");
    setPublicPayload(null);
    try {
      let path = `/api/match/${publicTargetId.trim()}`;
      if (publicTab === "rankings") {
        const [type, id] = publicTargetId.split(":");
        path = fillRoute(ROUTE_TEMPLATES.publicRankings, { type: (type || "").trim(), id: (id || "").trim() });
      }
      if (publicTab === "tournament-fixtures") path = `/api/public/tournaments/${publicTargetId.trim()}/fixtures`;
      if (publicTab === "tournament-standings") path = `/api/public/tournaments/${publicTargetId.trim()}/standings`;
      if (publicTab === "championship") path = `/api/public/championships/${publicTargetId.trim()}`;
      if (publicTab === "championship-fixtures") path = `/api/public/championships/${publicTargetId.trim()}/fixtures`;
      if (publicTab === "championship-stats") path = `/api/public/championships/${publicTargetId.trim()}/stats`;

      const response = await fetch(`http://localhost:3000${path}`);
      const data = await response.json().catch(() => ({}));
      if (!response.ok) throw new Error(data.error || "Failed to load public data");
      setPublicPayload(data);
    } catch (err) {
      setMessageType("danger");
      setMessage(err.message || "Failed to load public data");
    }
  }

  async function checkViewerRoute() {
    try {
      let routePath = "/viewer";
      if (publicTab === "viewer-match") routePath = `/viewer/match/${publicTargetId.trim()}`;
      if (publicTab === "viewer-overview") routePath = `/viewer/match/${publicTargetId.trim()}/overview`;
      if (publicTab === "viewer-tournament") routePath = `/viewer/tournament/${publicTargetId.trim()}`;
      if (publicTab === "viewer-championship") routePath = `/viewer/championship/${publicTargetId.trim()}`;
      if (publicTab === "rankings-page") {
        const [type, id] = publicTargetId.split(":");
        routePath = `/rankings/${(type || "").trim()}/${(id || "").trim()}`;
      }
      const response = await fetch(`http://localhost:3000${routePath}`);
      const text = await response.text();
      if (!response.ok) throw new Error(`Failed to open route ${routePath}`);
      setMessageType("info");
      setMessage(`Route ${routePath} responded, length ${text.length}`);
    } catch (err) {
      setMessageType("danger");
      setMessage(err.message || "Failed to check route");
    }
  }

  async function loadInviteDetails() {
    if (!inviteToken.trim()) return;
    try {
      const detailsTemplate = inviteTarget === "team" ? ROUTE_TEMPLATES.teamInviteDetails : ROUTE_TEMPLATES.eventInviteDetails;
      const response = await fetch(`http://localhost:3000${fillRoute(detailsTemplate, { token: inviteToken.trim() })}`);
      const data = await response.json().catch(() => ({}));
      if (!response.ok) throw new Error(data.error || "Failed to load invite details");
      setPublicPayload(data);
      setMessageType("success");
      setMessage("Invite details loaded");
    } catch (err) {
      setMessageType("danger");
      setMessage(err.message || "Failed to load invite details");
    }
  }

  async function runInviteAction(action) {
    if (!inviteToken.trim()) return;
    try {
      const actionTemplate = inviteTarget === "team"
        ? (action === "accept" ? ROUTE_TEMPLATES.teamInviteAccept : ROUTE_TEMPLATES.teamInviteClaim)
        : (action === "accept" ? ROUTE_TEMPLATES.eventInviteAccept : ROUTE_TEMPLATES.eventInviteClaim);
      const response = await authClient.apiFetch(fillRoute(actionTemplate, { token: inviteToken.trim() }), { method: "POST" });
      const data = await response.json().catch(() => ({}));
      if (!response.ok) throw new Error(data.error || `Failed to ${action} invite`);
      setMessageType("success");
      setMessage(data.message || `Invite ${action} successful`);
    } catch (err) {
      setMessageType("danger");
      setMessage(err.message || `Failed to ${action} invite`);
    }
  }

  async function doRefresh() {
    try {
      await authClient.refresh();
      await reloadSession();
      setMessageType("success");
      setMessage("Token refreshed");
    } catch (err) {
      setMessageType("danger");
      setMessage(err.message || "Refresh failed");
    }
  }

  async function doLogout() {
    await authClient.logout();
    await reloadSession();
    setMessageType("info");
    setMessage("Logged out");
  }

  async function doLogoutAll() {
    try {
      await authClient.logoutAll();
    } finally {
      await reloadSession();
    }
  }

  const isLoggedIn = Boolean(session.token);
  const role = normalizeRole(session.role || "player");

  async function loadProfile() {
    try {
      const response = await authClient.apiFetch("/api/me/profile");
      const data = await response.json().catch(() => ({}));
      if (!response.ok) {
        throw new Error(data.error || "Failed to load profile");
      }
      setProfile(data);
      setShowProfile(true);
    } catch (error) {
      setMessageType("danger");
      setMessage(error.message || "Failed to load profile");
    }
  }

  const panel = useMemo(() => {
    if (role === "team_owner") return <OwnerPanel />;
    if (role === "organizer") return <OrganizerPanel />;
    return <PlayerPanel />;
  }, [role]);

  return (
    <SafeAreaView style={{ flex: 1, backgroundColor: "#0f172a" }}>
      {!isLoggedIn ? (
        <ScrollView contentContainerStyle={{ padding: 16 }}>
          <Text style={{ fontSize: 26, fontWeight: "700", color: "#fbbf24", marginBottom: 12 }}>RaidX Mobile Auth</Text>
          <StatusBox message={message} type={messageType} />
          <TabRow
            current={authMode}
            onChange={setAuthMode}
            items={[
              { value: "login", label: "Login" },
              { value: "signup", label: "Signup" }
            ]}
          />
          <TextInput
            placeholder="Username"
            placeholderTextColor="#94a3b8"
            value={identifier}
            onChangeText={setIdentifier}
            style={{ borderWidth: 1, borderColor: "#475569", color: "#fff", padding: 10, marginBottom: 8, borderRadius: 8 }}
          />
          {authMode === "signup" ? (
            <>
              <TextInput
                placeholder="Full name"
                placeholderTextColor="#94a3b8"
                value={fullName}
                onChangeText={setFullName}
                style={{ borderWidth: 1, borderColor: "#475569", color: "#fff", padding: 10, marginBottom: 8, borderRadius: 8 }}
              />
              <TextInput
                placeholder="Email"
                placeholderTextColor="#94a3b8"
                value={email}
                onChangeText={setEmail}
                style={{ borderWidth: 1, borderColor: "#475569", color: "#fff", padding: 10, marginBottom: 8, borderRadius: 8 }}
              />
              <TabRow
                current={signupRole}
                onChange={setSignupRole}
                items={[
                  { value: "player", label: "player" },
                  { value: "team_owner", label: "owner" },
                  { value: "organizer", label: "organizer" }
                ]}
              />
              {signupRole === "player" ? (
                <TextInput
                  placeholder="Position"
                  placeholderTextColor="#94a3b8"
                  value={position}
                  onChangeText={setPosition}
                  style={{ borderWidth: 1, borderColor: "#475569", color: "#fff", padding: 10, marginBottom: 8, borderRadius: 8 }}
                />
              ) : null}
            </>
          ) : null}
          <TextInput
            placeholder="Password"
            placeholderTextColor="#94a3b8"
            secureTextEntry
            value={password}
            onChangeText={setPassword}
            style={{ borderWidth: 1, borderColor: "#475569", color: "#fff", padding: 10, marginBottom: 8, borderRadius: 8 }}
          />
          {authMode === "signup" ? (
            <TextInput
              placeholder="Confirm password"
              placeholderTextColor="#94a3b8"
              secureTextEntry
              value={confirmPassword}
              onChangeText={setConfirmPassword}
              style={{ borderWidth: 1, borderColor: "#475569", color: "#fff", padding: 10, marginBottom: 8, borderRadius: 8 }}
            />
          ) : null}
          <Button label={authMode === "login" ? "Sign In" : "Create Account"} variant="primary" onPress={authMode === "login" ? doLogin : doSignup} />

          <SectionCard title="Public Viewer + Invite Tools" subtitle="Covers remaining public routes and invite token APIs.">
            <TabRow
              current={publicTab}
              onChange={setPublicTab}
              items={[
                { value: "match", label: "api/match" },
                { value: "rankings", label: "api/rankings" },
                { value: "tournament-fixtures", label: "tour fixtures" },
                { value: "tournament-standings", label: "tour standings" },
                { value: "championship", label: "champ" },
                { value: "championship-fixtures", label: "champ fix" },
                { value: "championship-stats", label: "champ stats" },
                { value: "viewer-match", label: "page match" },
                { value: "viewer-overview", label: "page overview" },
                { value: "viewer-tournament", label: "page tour" },
                { value: "viewer-championship", label: "page champ" },
                { value: "rankings-page", label: "page rankings" }
              ]}
            />
            <TextInput
              placeholder="Target ID (rankings: type:id)"
              placeholderTextColor="#94a3b8"
              value={publicTargetId}
              onChangeText={setPublicTargetId}
              style={{ borderWidth: 1, borderColor: "#475569", color: "#fff", padding: 10, marginBottom: 8, borderRadius: 8 }}
            />
            <Button label="Load Public API" onPress={loadPublicData} />
            <Button label="Check Viewer/Ranking Page Route" onPress={checkViewerRoute} />

            <TabRow
              current={inviteTarget}
              onChange={setInviteTarget}
              items={[
                { value: "team", label: "team token" },
                { value: "event", label: "event token" }
              ]}
            />
            <TextInput
              placeholder="Invite token"
              placeholderTextColor="#94a3b8"
              value={inviteToken}
              onChangeText={setInviteToken}
              style={{ borderWidth: 1, borderColor: "#475569", color: "#fff", padding: 10, marginBottom: 8, borderRadius: 8 }}
            />
            <Button label="Load Invite Details" onPress={loadInviteDetails} />
            <Button label="Accept Invite Token" onPress={() => runInviteAction("accept")} />
            <Button label="Claim Invite Token" onPress={() => runInviteAction("claim")} />

            {publicPayload ? <Text style={{ color: "#cbd5e1" }}>{JSON.stringify(publicPayload)}</Text> : null}
          </SectionCard>
        </ScrollView>
      ) : (
        <ScrollView contentContainerStyle={{ padding: 16 }}>
          <View style={{ backgroundColor: "#020617", borderRadius: 10, padding: 12, marginBottom: 12, borderWidth: 1, borderColor: "#334155" }}>
            <Text style={{ color: "#f97316", fontSize: 20, fontWeight: "800" }}>⚡ RaidX</Text>
            <Text style={{ color: "#fbbf24", marginTop: 4, marginBottom: 6 }}>Role: {role.replace("_", " ")}</Text>
            <Text style={{ color: "#cbd5e1" }}>User ID: {session.userId || "-"}</Text>
            <View style={{ marginTop: 10 }}>
              <Button label="My Profile" onPress={loadProfile} />
              <Button label="Refresh Token" onPress={doRefresh} />
              <Button label="Logout" onPress={doLogout} />
              <Button label="Logout All Devices" onPress={doLogoutAll} />
            </View>
          </View>
          <StatusBox message={message} type={messageType} />
          {showProfile && profile ? (
            <SectionCard title="My Profile" subtitle="RBAC account details">
              <ListItem title={profile.fullName || "Unknown"}>
                <Text style={{ color: "#cbd5e1" }}>Email: {profile.email || "-"}</Text>
                <Text style={{ color: "#cbd5e1" }}>User ID: {profile.userId || "-"}</Text>
                <Text style={{ color: "#cbd5e1" }}>Role: {normalizeRole(profile.role || role)}</Text>
                <Text style={{ color: "#cbd5e1" }}>Position: {profile.position || "-"}</Text>
                <Text style={{ color: "#cbd5e1" }}>Total Points: {profile.totalPoints || 0}</Text>
                <Text style={{ color: "#cbd5e1" }}>Raid Points: {profile.raidPoints || 0}</Text>
                <Text style={{ color: "#cbd5e1" }}>Defence Points: {profile.defencePoints || 0}</Text>
                <Text style={{ color: "#cbd5e1" }}>Matches Played: {profile.matchesPlayed || 0}</Text>
              </ListItem>
              <Button label="Hide Profile" onPress={() => setShowProfile(false)} />
            </SectionCard>
          ) : null}
          {panel}
        </ScrollView>
      )}
    </SafeAreaView>
  );
}
