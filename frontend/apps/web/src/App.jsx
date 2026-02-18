import { useEffect, useMemo, useState } from "react";
import { Navigate, Route, Routes, useNavigate, useParams } from "react-router-dom";
import { authClient } from "./authClient";

function LoginPage() {
  const [identifier, setIdentifier] = useState("");
  const [password, setPassword] = useState("");
  const [error, setError] = useState("");
  const [loading, setLoading] = useState(false);
  const navigate = useNavigate();

  useEffect(() => {
    authClient.getSessionSummary().then((summary) => {
      if (summary.token) {
        navigate(`/dashboard/${normalizeRole(summary.role || "player")}`, { replace: true });
      }
    });
  }, [navigate]);

  async function onSubmit(event) {
    event.preventDefault();
    setError("");
    setLoading(true);
    try {
      const data = await authClient.login({ identifier, password });
      const role = normalizeRole(data.role || "player");
      navigate(`/dashboard/${role}`);
    } catch (err) {
      setError(err.message || "Login failed");
    } finally {
      setLoading(false);
    }
  }

  return (
    <div className="login-page">
      <div className="floating-bg circle1" />
      <div className="floating-bg circle2" />
      <div className="login-card">
        <h1>Login to RaidX</h1>
        <form onSubmit={onSubmit} className="form-grid">
          <label>Username or Email</label>
          <input
            placeholder="Enter your username or email"
            value={identifier}
            onChange={(event) => setIdentifier(event.target.value)}
            autoComplete="username"
          />
          <label>Password</label>
          <input
            type="password"
            placeholder="Enter your password"
            value={password}
            onChange={(event) => setPassword(event.target.value)}
            autoComplete="current-password"
          />
          {error ? <div className="error-box">{error}</div> : null}
          <button className="btn-primary-orange" disabled={loading} type="submit">
            {loading ? "Signing in..." : "Login"}
          </button>
          <button className="btn-ghost-link" type="button" onClick={() => window.location.assign("/")}>← Back to Home</button>
        </form>
      </div>
    </div>
  );
}

function DashboardPage({ role }) {
  const [session, setSession] = useState({ token: null, userId: null, role: null, exp: null });
  const [statusMessage, setStatusMessage] = useState("");
  const [statusType, setStatusType] = useState("info");
  const navigate = useNavigate();

  useEffect(() => {
    authClient.getSessionSummary().then((summary) => {
      setSession(summary);
      if (!summary.token) {
        navigate("/login");
        return;
      }

      if (normalizeRole(summary.role || "player") !== role) {
        navigate(`/dashboard/${normalizeRole(summary.role || "player")}`, { replace: true });
      }
    });
  }, [navigate, role]);

  async function doRefresh() {
    setStatusMessage("");
    try {
      await authClient.refresh();
      const summary = await authClient.getSessionSummary();
      setSession(summary);
      setStatusType("success");
      setStatusMessage("Token refreshed");
    } catch (err) {
      setStatusType("danger");
      setStatusMessage(err.message || "Refresh failed");
      navigate("/login");
    }
  }

  async function doLogout() {
    await authClient.logout();
    navigate("/login");
  }

  async function doLogoutAll() {
    try {
      await authClient.logoutAll();
    } finally {
      navigate("/login");
    }
  }

  const roleTitle = role === "team_owner" ? "Team Owner" : role === "organizer" ? "Organizer" : "Player";
  return (
    <div className="dashboard-page">
      <header className="dashboard-navbar">
        <div className="brand">⚡ RaidX</div>
        <div className="navbar-actions">
          <span className="role-badge">{roleTitle}</span>
          <button className="btn-outline" onClick={() => window.location.assign("/profile")}>My Profile</button>
          <button className="btn-outline" onClick={() => window.location.assign("/viewer")}>View Score</button>
          <button className="btn-outline" onClick={doRefresh}>Refresh</button>
          <button className="btn-outline" onClick={doLogoutAll}>Logout All</button>
          <button className="btn-outline" onClick={doLogout}>Logout</button>
        </div>
      </header>

      <main className="dashboard-container">
        {statusMessage ? <div className={`status-box ${statusType}`}>{statusMessage}</div> : null}
        <section className="dashboard-header">
          <h1>{roleTitle} Dashboard</h1>
          <p>Signed in as {session.userId || "-"}</p>
        </section>

        {role === "team_owner" ? <OwnerDashboard /> : null}
        {role === "organizer" ? <OrganizerDashboard /> : null}
        {role === "player" ? <PlayerDashboard /> : null}
        {!isSupportedRole(role) ? (
          <div className="panel-card">
            <h3>Unsupported role</h3>
            <p>The role from session is not mapped yet.</p>
          </div>
        ) : null}
      </main>
    </div>
  );
}

function PlayerDashboard() {
  const [inviteTab, setInviteTab] = useState("pending");
  const [infoTab, setInfoTab] = useState("teams");
  const [invites, setInvites] = useState({ pending: [], accepted: [], declined: [] });
  const [teams, setTeams] = useState([]);
  const [events, setEvents] = useState([]);
  const [loading, setLoading] = useState(false);
  const [statusMessage, setStatusMessage] = useState("");
  const [statusType, setStatusType] = useState("info");

  async function loadPlayerData() {
    setLoading(true);
    setStatusMessage("");
    try {
      const [invitesRes, teamsRes, eventsRes] = await Promise.all([
        authClient.apiFetch("/api/invitations"),
        authClient.apiFetch("/api/player/teams"),
        authClient.apiFetch("/api/player/events")
      ]);

      const inviteData = await invitesRes.json();
      const allInvites = Array.isArray(inviteData) ? inviteData : [];
      const pending = allInvites.filter((invite) => ["pending", "invited_via_link"].includes(normalizeStatus(invite)));
      const accepted = allInvites.filter((invite) => ["accepted", "accepted_by_owner"].includes(normalizeStatus(invite)));
      const declined = allInvites.filter((invite) => ["declined", "declined_by_owner"].includes(normalizeStatus(invite)));

      setInvites({
        pending,
        accepted,
        declined
      });
      setTeams(safeArray(await teamsRes.json()));
      setEvents(safeArray(await eventsRes.json()));
    } catch (error) {
      setStatusType("danger");
      setStatusMessage(error.message || "Failed to load player dashboard data");
    } finally {
      setLoading(false);
    }
  }

  useEffect(() => {
    loadPlayerData();
  }, []);

  async function updateInvite(invitationId, status) {
    try {
      const response = await authClient.apiFetch(`/api/invitations/${invitationId}`, {
        method: "PUT",
        body: JSON.stringify({ status })
      });
      const data = await response.json().catch(() => ({}));
      if (!response.ok) {
        throw new Error(data.error || "Failed to update invitation");
      }
      await loadPlayerData();
    } catch (error) {
      setStatusType("danger");
      setStatusMessage(error.message || "Failed to update invitation");
    }
  }

  const currentInvites = invites[inviteTab] || [];

  return (
    <>
      <section className="panel-card">
        {statusMessage ? <div className={`status-box ${statusType}`}>{statusMessage}</div> : null}
        <div className="panel-head">
          <div>
            <h3>Team Invitations</h3>
            <p>Review and manage incoming team requests.</p>
          </div>
          <button className="btn-primary-orange" onClick={loadPlayerData}>{loading ? "Loading..." : "Refresh"}</button>
        </div>
        <div className="tab-row">
          {renderTab(inviteTab, setInviteTab, "pending", `Pending (${invites.pending.length})`)}
          {renderTab(inviteTab, setInviteTab, "accepted", `Accepted (${invites.accepted.length})`)}
          {renderTab(inviteTab, setInviteTab, "declined", `Declined (${invites.declined.length})`)}
        </div>
        <div className="list-wrap">
          {!currentInvites.length ? <div className="empty-state">No invitations in this tab.</div> : null}
          {currentInvites.map((invite) => (
            <article key={getInviteId(invite)} className="request-card">
              <h4>{invite.teamName || "Team invite"}</h4>
              <p>Owner: {invite.ownerName || "Unknown"}</p>
              <p>Invite ID: {getInviteId(invite)}</p>
              <p>Team ID: {invite.teamId || "-"}</p>
              <p>Status: <span className="badge-custom">{getStatusDisplay(normalizeStatus(invite))}</span></p>
              {getDeclineReason(invite) && normalizeStatus(invite).startsWith("declined") ? <p>Reason: {getDeclineReason(invite)}</p> : null}
              {inviteTab === "pending" && normalizeStatus(invite) !== "invited_via_link" ? (
                <div className="card-actions">
                  <button className="btn-primary-orange" onClick={() => updateInvite(getInviteId(invite), "accepted")}>Accept</button>
                  <button className="btn-outline" onClick={() => updateInvite(getInviteId(invite), "declined")}>Decline</button>
                </div>
              ) : null}
            </article>
          ))}
        </div>
      </section>

      <section className="panel-card">
        <div className="panel-head">
          <div>
            <h3>My Teams & Events</h3>
            <p>Teams you belong to and recent events.</p>
          </div>
          <button className="btn-primary-orange" onClick={loadPlayerData}>{loading ? "Loading..." : "Refresh"}</button>
        </div>
        <div className="tab-row">
          {renderTab(infoTab, setInfoTab, "teams", `My Teams (${teams.length})`)}
          {renderTab(infoTab, setInfoTab, "events", `My Events (${events.length})`)}
        </div>
        <div className="list-wrap">
          {infoTab === "teams" ? teams.map((team) => (
            <article key={team.teamId || team.TeamID} className="entity-card">
              <h4>{team.teamName || team.TeamName}</h4>
              <p>Status: <span className="badge-custom">{team.status || team.Status}</span></p>
            </article>
          )) : events.map((event) => (
            <article key={`${event.eventType}-${event.eventId || event.eventName}`} className="entity-card">
              <h4>{event.eventName || "Event"}</h4>
              <p>Type: <span className="badge-custom">{event.eventType}</span></p>
              <p>Matches: {event.matchCount || 0}</p>
            </article>
          ))}
          {infoTab === "teams" && !teams.length ? <div className="empty-state">No teams yet.</div> : null}
          {infoTab === "events" && !events.length ? <div className="empty-state">No events yet.</div> : null}
        </div>
      </section>
    </>
  );
}

function OwnerDashboard() {
  const [tab, setTab] = useState("teams");
  const [inviteTab, setInviteTab] = useState("pending");
  const [teams, setTeams] = useState([]);
  const [invites, setInvites] = useState({ pending: [], accepted: [], declined: [] });
  const [selectedTeamByInvite, setSelectedTeamByInvite] = useState({});
  const [creating, setCreating] = useState(false);
  const [teamName, setTeamName] = useState("");
  const [statusMessage, setStatusMessage] = useState("");
  const [statusType, setStatusType] = useState("info");

  async function loadOwnerData() {
    setStatusMessage("");
    try {
      const [teamsRes, invitesRes] = await Promise.all([
        authClient.apiFetch("/api/owner/teams"),
        authClient.apiFetch("/api/owner/event-invitations")
      ]);

      const teamData = safeArray(await teamsRes.json());
      const inviteData = safeArray(await invitesRes.json());
      const pending = inviteData.filter((invite) => ["pending", "invited_via_link"].includes(normalizeStatus(invite)));
      const accepted = inviteData.filter((invite) => ["accepted", "accepted_by_owner", "accepted_by_organizer"].includes(normalizeStatus(invite)));
      const declined = inviteData.filter((invite) => ["declined", "declined_by_organizer"].includes(normalizeStatus(invite)));

      setTeams(teamData);
      setInvites({ pending, accepted, declined });
    } catch (error) {
      setStatusType("danger");
      setStatusMessage(error.message || "Failed to load owner dashboard data");
    }
  }

  useEffect(() => {
    loadOwnerData();
  }, []);

  async function handleCreateTeam(event) {
    event.preventDefault();
    if (!teamName.trim()) return;
    setCreating(true);
    try {
      const response = await authClient.apiFetch("/api/teams", {
        method: "POST",
        body: JSON.stringify({ team_name: teamName.trim() })
      });
      const data = await response.json().catch(() => ({}));
      if (!response.ok) {
        throw new Error(data.error || "Failed to create team");
      }
      setTeamName("");
      setTab("teams");
      setStatusType("success");
      setStatusMessage(`Team created: ${data.team_name || teamName}`);
      await loadOwnerData();
    } catch (error) {
      setStatusType("danger");
      setStatusMessage(error.message || "Failed to create team");
    } finally {
      setCreating(false);
    }
  }

  async function updateInvite(invitation, status) {
    const payload = { status };
    const inviteId = getInviteId(invitation);

    if (status === "accepted") {
      const selectedTeamId = selectedTeamByInvite[inviteId] || invitation.teamId || "";
      if (!selectedTeamId) {
        setStatusType("warning");
        setStatusMessage("Select a team before accepting.");
        return;
      }
      const selectedTeam = teams.find((team) => getTeamId(team) === selectedTeamId);
      const playersCount = Number(selectedTeam?.Players || selectedTeam?.players || 0);
      if (playersCount < 7) {
        setStatusType("warning");
        setStatusMessage("Selected team must have at least 7 players.");
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
      if (!response.ok) {
        throw new Error(data.error || "Failed to update invitation");
      }
      await loadOwnerData();
      setStatusType("success");
      setStatusMessage("Invitation updated");
    } catch (error) {
      setStatusType("danger");
      setStatusMessage(error.message || "Failed to update invitation");
    }
  }

  const selectedInvites = invites[inviteTab] || [];

  return (
    <section className="panel-card">
      {statusMessage ? <div className={`status-box ${statusType}`}>{statusMessage}</div> : null}
      <div className="tab-row">
        {renderTab(tab, setTab, "teams", `My Teams (${teams.length})`)}
        {renderTab(tab, setTab, "invites", "Event Invitations")}
        {renderTab(tab, setTab, "create", "Create Team")}
      </div>

      {tab === "teams" ? (
        <div className="list-wrap">
          {!teams.length ? <div className="empty-state">No teams found.</div> : null}
          {teams.map((team) => (
            <article key={team.ID || team.id} className="entity-card">
              <h4>{team.TeamName || team.teamName}</h4>
              <p>Players: {typeof team.Players === "number" ? team.Players : "-"}</p>
              <p>Status: <span className="badge-custom">{team.Status || team.status || "active"}</span></p>
            </article>
          ))}
        </div>
      ) : null}

      {tab === "invites" ? (
        <>
          <div className="tab-row compact">
            {renderTab(inviteTab, setInviteTab, "pending", `Pending (${invites.pending.length})`)}
            {renderTab(inviteTab, setInviteTab, "accepted", `Accepted (${invites.accepted.length})`)}
            {renderTab(inviteTab, setInviteTab, "declined", `Declined (${invites.declined.length})`)}
          </div>
          <div className="list-wrap">
            {!selectedInvites.length ? <div className="empty-state">No invitations in this tab.</div> : null}
            {selectedInvites.map((invite) => (
              <article key={getInviteId(invite)} className="request-card">
                <h4>{invite.eventName || "Event Invite"}</h4>
                <p>Team Owner: {invite.ownerName || "Unknown"}</p>
                <p>Team: {invite.teamName || "Will use your first team if accepted"}</p>
                <p>Status: <span className="badge-custom">{getStatusDisplay(normalizeStatus(invite))}</span></p>
                {getDeclineReason(invite) && normalizeStatus(invite).startsWith("declined") ? <p>Reason: {getDeclineReason(invite)}</p> : null}
                {inviteTab === "pending" && normalizeStatus(invite) !== "invited_via_link" ? (
                  <div className="card-actions">
                    <select
                      className="inline-select"
                      value={selectedTeamByInvite[getInviteId(invite)] || ""}
                      onChange={(event) => setSelectedTeamByInvite((prev) => ({
                        ...prev,
                        [getInviteId(invite)]: event.target.value
                      }))}
                    >
                      <option value="">Select team to accept...</option>
                      {teams.map((team) => {
                        const playersCount = Number(team.Players || team.players || 0);
                        const teamId = getTeamId(team);
                        return (
                          <option key={teamId} value={teamId} disabled={playersCount < 7}>
                            {(team.TeamName || team.teamName || "Team")} ({playersCount} players)
                          </option>
                        );
                      })}
                    </select>
                    <button className="btn-primary-orange" onClick={() => updateInvite(invite, "accepted")}>Accept</button>
                    <button className="btn-outline" onClick={() => updateInvite(invite, "declined")}>Decline</button>
                  </div>
                ) : null}
              </article>
            ))}
          </div>
        </>
      ) : null}

      {tab === "create" ? (
        <form className="form-grid create-form" onSubmit={handleCreateTeam}>
          <label>Team Name</label>
          <input value={teamName} onChange={(event) => setTeamName(event.target.value)} placeholder="Enter team name" />
          <button className="btn-primary-orange" type="submit" disabled={creating}>{creating ? "Creating..." : "Create Team"}</button>
        </form>
      ) : null}
    </section>
  );
}

function OrganizerDashboard() {
  const [tab, setTab] = useState("events");
  const [eventFilter, setEventFilter] = useState("ongoing");
  const [inviteFilter, setInviteFilter] = useState("pending");
  const [events, setEvents] = useState([]);
  const [invites, setInvites] = useState({ pending: [], accepted: [], declined: [] });
  const [createModel, setCreateModel] = useState({ event_name: "", event_type: "match", max_teams: 4 });
  const [statusMessage, setStatusMessage] = useState("");
  const [statusType, setStatusType] = useState("info");

  async function loadOrganizerData() {
    setStatusMessage("");
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
      setStatusType("danger");
      setStatusMessage(error.message || "Failed to load organizer dashboard data");
    }
  }

  useEffect(() => {
    loadOrganizerData();
  }, []);

  const filteredEvents = useMemo(() => {
    return events.filter((event) => {
      const status = (event.status || "").toLowerCase();
      if (eventFilter === "ongoing") return status === "active" || status === "ongoing";
      if (eventFilter === "pending") return status !== "completed" && status !== "active" && status !== "ongoing";
      if (eventFilter === "completed") return status === "completed";
      return true;
    });
  }, [events, eventFilter]);

  async function createEvent(event) {
    event.preventDefault();
    if (!createModel.event_name.trim()) {
      setStatusType("warning");
      setStatusMessage("Event name is required");
      return;
    }
    if (["tournament", "championship"].includes(createModel.event_type) && Number(createModel.max_teams) < 4) {
      setStatusType("warning");
      setStatusMessage("Number of teams must be at least 4 for tournament/championship");
      return;
    }

    const payload = {
      event_name: createModel.event_name,
      event_type: createModel.event_type,
      max_teams: Number(createModel.max_teams || 0)
    };
    try {
      const response = await authClient.apiFetch("/api/events", {
        method: "POST",
        body: JSON.stringify(payload)
      });
      const data = await response.json().catch(() => ({}));
      if (!response.ok) {
        throw new Error(data.error || "Failed to create event");
      }
      setCreateModel({ event_name: "", event_type: "match", max_teams: 4 });
      setTab("events");
      setStatusType("success");
      setStatusMessage(`Event created: ${data.event_name || payload.event_name}`);
      await loadOrganizerData();
    } catch (error) {
      setStatusType("danger");
      setStatusMessage(error.message || "Failed to create event");
    }
  }

  return (
    <section className="panel-card">
      {statusMessage ? <div className={`status-box ${statusType}`}>{statusMessage}</div> : null}
      <div className="tab-row">
        {renderTab(tab, setTab, "events", "My Events")}
        {renderTab(tab, setTab, "invites", "Invite Responses")}
        {renderTab(tab, setTab, "create", "Create Event")}
      </div>

      {tab === "events" ? (
        <>
          <div className="tab-row compact">
            {renderTab(eventFilter, setEventFilter, "ongoing", "Ongoing")}
            {renderTab(eventFilter, setEventFilter, "pending", "Pending")}
            {renderTab(eventFilter, setEventFilter, "completed", "Completed")}
          </div>
          <div className="list-wrap">
            {!filteredEvents.length ? <div className="empty-state">No events found in this state.</div> : null}
            {filteredEvents.map((event) => (
              <article key={event.id} className="entity-card clickable" onClick={() => window.location.assign(`/organizer/event/${event.id}`)}>
                <h4>{event.eventName}</h4>
                <p>Type: <span className="badge-custom">{event.eventType}</span></p>
                <p>Status: <span className="badge-custom">{event.status}</span></p>
                <p>Accepted: {event.counts?.accepted || 0} | Pending: {event.counts?.pending || 0}</p>
              </article>
            ))}
          </div>
        </>
      ) : null}

      {tab === "invites" ? (
        <>
          <div className="tab-row compact">
            {renderTab(inviteFilter, setInviteFilter, "pending", `Pending (${invites.pending.length})`)}
            {renderTab(inviteFilter, setInviteFilter, "accepted", `Accepted (${invites.accepted.length})`)}
            {renderTab(inviteFilter, setInviteFilter, "declined", `Declined (${invites.declined.length})`)}
          </div>
          <div className="list-wrap">
            {!(invites[inviteFilter] || []).length ? <div className="empty-state">No invite responses in this tab.</div> : null}
            {(invites[inviteFilter] || []).map((invite) => (
              <article key={getInviteId(invite)} className="request-card">
                <h4>{invite.eventName || "Event Invite"}</h4>
                <p>Owner: {invite.ownerName || "Unknown"}</p>
                <p>Team: {invite.teamName || "Unassigned"}</p>
                <p>Status: <span className="badge-custom">{getStatusDisplay(normalizeStatus(invite))}</span></p>
                {getDeclineReason(invite) ? <p>Reason: {getDeclineReason(invite)}</p> : null}
              </article>
            ))}
          </div>
        </>
      ) : null}

      {tab === "create" ? (
        <form className="form-grid create-form" onSubmit={createEvent}>
          <label>Event Name</label>
          <input
            value={createModel.event_name}
            onChange={(event) => setCreateModel((prev) => ({ ...prev, event_name: event.target.value }))}
            placeholder="Event name"
            required
          />
          <label>Event Type</label>
          <select
            value={createModel.event_type}
            onChange={(event) => setCreateModel((prev) => ({ ...prev, event_type: event.target.value }))}
          >
            <option value="match">Match</option>
            <option value="tournament">Tournament</option>
            <option value="championship">Championship</option>
          </select>
          {createModel.event_type !== "match" ? (
            <>
              <label>No. of Teams</label>
              <input
                type="number"
                min={4}
                value={createModel.max_teams}
                onChange={(event) => setCreateModel((prev) => ({ ...prev, max_teams: event.target.value }))}
              />
            </>
          ) : null}
          <button className="btn-primary-orange" type="submit">Create Event</button>
        </form>
      ) : null}
    </section>
  );
}

function normalizeStatus(invite) {
  return String(invite?.status || invite?.Status || "pending").toLowerCase();
}

function getDeclineReason(invite) {
  return invite?.declineReason || invite?.decline_reason || invite?.DeclineReason || "";
}

function getInviteId(invite) {
  return invite?.id || invite?.ID || invite?._id || "";
}

function getTeamId(team) {
  return team?.ID || team?.id || team?.teamId || team?.TeamID || "";
}

function safeArray(data) {
  return Array.isArray(data) ? data : [];
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

function ProfilePage() {
  const [profile, setProfile] = useState(null);
  const [error, setError] = useState("");
  const [loading, setLoading] = useState(true);
  const navigate = useNavigate();

  useEffect(() => {
    async function loadProfile() {
      try {
        const response = await authClient.apiFetch("/api/me/profile");
        const data = await response.json().catch(() => ({}));
        if (!response.ok) {
          throw new Error(data.error || "Failed to load profile");
        }
        setProfile(data);
      } catch (err) {
        setError(err.message || "Failed to load profile");
      } finally {
        setLoading(false);
      }
    }

    loadProfile();
  }, []);

  if (loading) {
    return <div className="dashboard-container"><div className="panel-card">Loading profile...</div></div>;
  }

  return (
    <div className="dashboard-page">
      <header className="dashboard-navbar">
        <div className="brand">⚡ RaidX</div>
        <div className="navbar-actions">
          <button className="btn-outline" onClick={() => navigate(-1)}>Back</button>
        </div>
      </header>
      <main className="dashboard-container">
        <section className="dashboard-header">
          <h1>My Profile</h1>
          <p>RBAC account details</p>
        </section>
        {error ? <div className="status-box danger">{error}</div> : null}
        {profile ? (
          <section className="panel-card">
            <div className="list-wrap">
              <article className="entity-card">
                <h4>{profile.fullName || "Unknown"}</h4>
                <p>Email: {profile.email || "-"}</p>
                <p>User ID: {profile.userId || "-"}</p>
                <p>Role: <span className="badge-custom">{normalizeRole(profile.role || "player")}</span></p>
                <p>Position: {profile.position || "-"}</p>
              </article>
              <article className="entity-card">
                <h4>Player Stats</h4>
                <p>Total Points: {profile.totalPoints || 0}</p>
                <p>Raid Points: {profile.raidPoints || 0}</p>
                <p>Defence Points: {profile.defencePoints || 0}</p>
                <p>Matches Played: {profile.matchesPlayed || 0}</p>
                <p>MVP: {profile.mvpCount || 0}</p>
              </article>
            </div>
          </section>
        ) : null}
      </main>
    </div>
  );
}

function renderTab(current, setCurrent, value, label) {
  return (
    <button
      className={`tab-btn ${current === value ? "active" : ""}`}
      onClick={() => setCurrent(value)}
      type="button"
    >
      {label}
    </button>
  );
}

function normalizeRole(role) {
  const normalized = String(role || "player").toLowerCase();
  if (normalized === "owner") return "team_owner";
  return normalized;
}

function isSupportedRole(role) {
  return role === "player" || role === "team_owner" || role === "organizer";
}

function SessionGate({ children }) {
  const [loading, setLoading] = useState(true);
  const [isValid, setIsValid] = useState(false);

  useEffect(() => {
    authClient
      .getToken()
      .then((token) => {
        setIsValid(Boolean(token));
      })
      .finally(() => setLoading(false));
  }, []);

  if (loading) return <div className="page">Loading...</div>;
  if (!isValid) return <Navigate to="/login" replace />;
  return children;
}

export default function App() {
  return (
    <Routes>
      <Route path="/login" element={<LoginPage />} />
      <Route
        path="/profile"
        element={
          <SessionGate>
            <ProfilePage />
          </SessionGate>
        }
      />
      <Route
        path="/dashboard/:role"
        element={
          <SessionGate>
            <DashboardRoleWrapper />
          </SessionGate>
        }
      />
      <Route path="*" element={<Navigate to="/login" replace />} />
    </Routes>
  );
}

function DashboardRoleWrapper() {
  const params = useParams();
  const role = normalizeRole(params.role || "player");
  return <DashboardPage role={role} />;
}
