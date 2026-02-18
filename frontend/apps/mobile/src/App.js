import { useEffect, useMemo, useState } from "react";
import { SafeAreaView, ScrollView, Text, TextInput, TouchableOpacity, View } from "react-native";
import { authClient } from "./authClient";

function Button({ label, onPress, variant = "light", disabled = false }) {
  const backgroundColor = variant === "primary" ? "#f97316" : variant === "danger" ? "#ef4444" : "#334155";
  return (
    <TouchableOpacity disabled={disabled} onPress={onPress} style={{ padding: 12, marginBottom: 8, backgroundColor, borderRadius: 8, opacity: disabled ? 0.65 : 1 }}>
      <Text style={{ color: "#fff", fontWeight: "600" }}>{label}</Text>
    </TouchableOpacity>
  );
}

function normalizeRole(role) {
  const normalized = String(role || "player").toLowerCase();
  if (normalized === "owner") return "team_owner";
  return normalized;
}

function normalizeStatus(invite) {
  return String(invite?.status || invite?.Status || "pending").toLowerCase();
}

function getInviteId(invite) {
  return invite?.id || invite?.ID || invite?._id || "";
}

function getDeclineReason(invite) {
  return invite?.declineReason || invite?.decline_reason || invite?.DeclineReason || "";
}

function getStatusDisplay(statusValue) {
  const normalized = String(statusValue || "pending").toLowerCase();
  const labels = {
    invited_via_link: "waiting for approval",
    accepted_by_owner: "accepted by owner",
    declined_by_owner: "declined by owner",
    accepted_by_organizer: "accepted by organizer",
    declined_by_organizer: "declined by organizer",
    pending: "pending",
    accepted: "accepted",
    declined: "declined"
  };
  return labels[normalized] || normalized;
}

function safeArray(data) {
  return Array.isArray(data) ? data : [];
}

function getTeamId(team) {
  return team?.ID || team?.id || team?.teamId || team?.TeamID || "";
}

function TabRow({ items, current, onChange }) {
  return (
    <View style={{ flexDirection: "row", flexWrap: "wrap", gap: 8, marginBottom: 12 }}>
      {items.map((item) => (
        <TouchableOpacity
          key={item.value}
          onPress={() => onChange(item.value)}
          style={{
            paddingVertical: 8,
            paddingHorizontal: 12,
            borderRadius: 999,
            backgroundColor: current === item.value ? "#f97316" : "#334155"
          }}
        >
          <Text style={{ color: "white", fontWeight: "600" }}>{item.label}</Text>
        </TouchableOpacity>
      ))}
    </View>
  );
}

function SectionCard({ title, subtitle, children }) {
  return (
    <View style={{ backgroundColor: "#1e293b", borderRadius: 12, borderWidth: 1, borderColor: "#334155", padding: 12, marginBottom: 12 }}>
      <Text style={{ color: "#fbbf24", fontSize: 18, fontWeight: "700" }}>{title}</Text>
      {subtitle ? <Text style={{ color: "#cbd5e1", marginBottom: 10 }}>{subtitle}</Text> : null}
      {children}
    </View>
  );
}

function StatusBox({ message, type = "info" }) {
  if (!message) return null;
  const palette = type === "success"
    ? { bg: "#14532d", border: "#22c55e", text: "#dcfce7" }
    : type === "warning"
      ? { bg: "#78350f", border: "#f59e0b", text: "#fef3c7" }
      : type === "danger"
        ? { bg: "#7f1d1d", border: "#ef4444", text: "#fecaca" }
        : { bg: "#1e3a8a", border: "#60a5fa", text: "#dbeafe" };
  return (
    <View style={{ backgroundColor: palette.bg, borderColor: palette.border, borderWidth: 1, borderRadius: 8, padding: 10, marginBottom: 10 }}>
      <Text style={{ color: palette.text }}>{message}</Text>
    </View>
  );
}

function ListItem({ title, children }) {
  return (
    <View style={{ backgroundColor: "#0f172a", borderColor: "#334155", borderWidth: 1, borderRadius: 10, padding: 10, marginBottom: 8 }}>
      <Text style={{ color: "#fbbf24", fontWeight: "700", marginBottom: 4 }}>{title}</Text>
      {children}
    </View>
  );
}

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
    } catch (error) {
      setMessageType("danger");
      setMessage(error.message || "Failed to load owner data");
    }
  }

  useEffect(() => {
    loadData();
  }, []);

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
    } catch (error) {
      setMessageType("danger");
      setMessage(error.message || "Failed to load organizer data");
    }
  }

  useEffect(() => {
    loadData();
  }, []);

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
    </SectionCard>
  );
}

export default function App() {
  const [identifier, setIdentifier] = useState("");
  const [password, setPassword] = useState("");
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
        <View style={{ flex: 1, justifyContent: "center", padding: 16 }}>
          <Text style={{ fontSize: 26, fontWeight: "700", color: "#fbbf24", marginBottom: 12 }}>RaidX Mobile Login</Text>
          <StatusBox message={message} type={messageType} />
          <TextInput
            placeholder="Username or email"
            placeholderTextColor="#94a3b8"
            value={identifier}
            onChangeText={setIdentifier}
            style={{ borderWidth: 1, borderColor: "#475569", color: "#fff", padding: 10, marginBottom: 8, borderRadius: 8 }}
          />
          <TextInput
            placeholder="Password"
            placeholderTextColor="#94a3b8"
            secureTextEntry
            value={password}
            onChangeText={setPassword}
            style={{ borderWidth: 1, borderColor: "#475569", color: "#fff", padding: 10, marginBottom: 8, borderRadius: 8 }}
          />
          <Button label="Sign In" variant="primary" onPress={doLogin} />
        </View>
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
