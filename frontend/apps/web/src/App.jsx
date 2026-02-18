import { useEffect, useMemo, useState } from "react";
import { Navigate, Route, Routes, useNavigate, useParams } from "react-router-dom";
import { ROUTE_TEMPLATES, fillRoute } from "@raidx/shared";
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
          <button className="btn-outline" type="button" onClick={() => navigate("/signup")}>Create Account</button>
          <button className="btn-ghost-link" type="button" onClick={() => window.location.assign("/")}>← Back to Home</button>
        </form>
      </div>
    </div>
  );
}

function SignupPage() {
  const [fullName, setFullName] = useState("");
  const [email, setEmail] = useState("");
  const [userId, setUserId] = useState("");
  const [password, setPassword] = useState("");
  const [confirmPassword, setConfirmPassword] = useState("");
  const [role, setRole] = useState("player");
  const [position, setPosition] = useState("raider");
  const [status, setStatus] = useState("");
  const [statusType, setStatusType] = useState("info");
  const [submitting, setSubmitting] = useState(false);
  const navigate = useNavigate();

  async function onSubmit(event) {
    event.preventDefault();
    setStatus("");
    setSubmitting(true);
    try {
      await authClient.signup({
        fullName,
        email,
        userId,
        password,
        confirmPassword,
        role,
        position
      });
      setStatusType("success");
      setStatus("Signup successful. Please login.");
      setTimeout(() => navigate("/login"), 800);
    } catch (error) {
      setStatusType("danger");
      setStatus(error.message || "Signup failed");
    } finally {
      setSubmitting(false);
    }
  }

  return (
    <div className="login-page">
      <div className="floating-bg circle1" />
      <div className="floating-bg circle2" />
      <div className="login-card">
        <h1>Create RaidX Account</h1>
        <form onSubmit={onSubmit} className="form-grid">
          <label>Full Name</label>
          <input value={fullName} onChange={(event) => setFullName(event.target.value)} required />
          <label>Email</label>
          <input type="email" value={email} onChange={(event) => setEmail(event.target.value)} required />
          <label>Username</label>
          <input value={userId} onChange={(event) => setUserId(event.target.value)} required />
          <label>Password</label>
          <input type="password" value={password} onChange={(event) => setPassword(event.target.value)} required />
          <label>Confirm Password</label>
          <input type="password" value={confirmPassword} onChange={(event) => setConfirmPassword(event.target.value)} required />
          <label>Role</label>
          <select value={role} onChange={(event) => setRole(event.target.value)}>
            <option value="player">Player</option>
            <option value="team_owner">Team Owner</option>
            <option value="organizer">Organizer</option>
          </select>
          {role === "player" ? (
            <>
              <label>Position</label>
              <input value={position} onChange={(event) => setPosition(event.target.value)} placeholder="raider/defender/all-rounder" />
            </>
          ) : null}
          {status ? <div className={`status-box ${statusType}`}>{status}</div> : null}
          <button className="btn-primary-orange" type="submit" disabled={submitting}>{submitting ? "Creating..." : "Sign Up"}</button>
          <button className="btn-outline" type="button" onClick={() => navigate("/login")}>Back to Login</button>
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
  const [selectedTeamId, setSelectedTeamId] = useState("");
  const [teamDetail, setTeamDetail] = useState(null);
  const [teamEditName, setTeamEditName] = useState("");
  const [teamEditCity, setTeamEditCity] = useState("");
  const [playerIdentifier, setPlayerIdentifier] = useState("");
  const [pendingApprovals, setPendingApprovals] = useState([]);
  const [inviteLinks, setInviteLinks] = useState([]);
  const [linkExpiresIn, setLinkExpiresIn] = useState("30d");
  const [linkMaxUses, setLinkMaxUses] = useState("");
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
  const [generatedLink, setGeneratedLink] = useState("");
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

      if (!selectedTeamId && teamData.length > 0) {
        setSelectedTeamId(getTeamId(teamData[0]));
      }
    } catch (error) {
      setStatusType("danger");
      setStatusMessage(error.message || "Failed to load owner dashboard data");
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

      if (!teamRes.ok) {
        throw new Error(teamData.error || "Failed to load team details");
      }
      if (!approvalsRes.ok) {
        throw new Error(approvalsData.error || "Failed to load pending approvals");
      }
      if (!linksRes.ok) {
        throw new Error(linksData.error || "Failed to load invite links");
      }

      setTeamDetail(teamData);
      setTeamEditName(teamData.TeamName || teamData.teamName || "");
      setTeamEditCity(teamData.City || teamData.city || "");
      setPendingApprovals(safeArray(approvalsData));
      setInviteLinks(safeArray(linksData.data).filter((link) => (link.targetId || "") === teamId));
    } catch (error) {
      setStatusType("danger");
      setStatusMessage(error.message || "Failed to load team context");
    }
  }

  useEffect(() => {
    loadOwnerData();
  }, []);

  useEffect(() => {
    if (!selectedTeamId) return;
    loadTeamContext(selectedTeamId);
  }, [selectedTeamId]);

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

  async function updateTeamDetails(event) {
    event.preventDefault();
    if (!selectedTeamId) return;

    try {
      const response = await authClient.apiFetch(`/api/teams/${selectedTeamId}`, {
        method: "PUT",
        body: JSON.stringify({ teamName: teamEditName, city: teamEditCity })
      });
      const data = await response.json().catch(() => ({}));
      if (!response.ok) {
        throw new Error(data.error || "Failed to update team");
      }
      setStatusType("success");
      setStatusMessage("Team updated successfully");
      await loadOwnerData();
      await loadTeamContext(selectedTeamId);
    } catch (error) {
      setStatusType("danger");
      setStatusMessage(error.message || "Failed to update team");
    }
  }

  async function addPlayerToSelectedTeam(event) {
    event.preventDefault();
    if (!selectedTeamId || !playerIdentifier.trim()) return;

    try {
      const response = await authClient.apiFetch(`/api/teams/${selectedTeamId}/add-player`, {
        method: "POST",
        body: JSON.stringify({ playerIdentifier: playerIdentifier.trim() })
      });
      const data = await response.json().catch(() => ({}));
      if (!response.ok) {
        throw new Error(data.error || "Failed to add player");
      }
      setPlayerIdentifier("");
      setStatusType("success");
      setStatusMessage("Player added");
      await loadOwnerData();
      await loadTeamContext(selectedTeamId);
    } catch (error) {
      setStatusType("danger");
      setStatusMessage(error.message || "Failed to add player");
    }
  }

  async function removePlayerFromSelectedTeam(playerId) {
    if (!selectedTeamId || !playerId) return;
    try {
      const response = await authClient.apiFetch(`/api/teams/${selectedTeamId}/remove-player/${playerId}`, {
        method: "DELETE"
      });
      const data = await response.json().catch(() => ({}));
      if (!response.ok) {
        throw new Error(data.error || "Failed to remove player");
      }
      setStatusType("success");
      setStatusMessage("Player removed");
      await loadOwnerData();
      await loadTeamContext(selectedTeamId);
    } catch (error) {
      setStatusType("danger");
      setStatusMessage(error.message || "Failed to remove player");
    }
  }

  async function deleteSelectedTeam() {
    if (!selectedTeamId) return;
    try {
      const response = await authClient.apiFetch(`/api/teams/${selectedTeamId}`, { method: "DELETE" });
      const data = await response.json().catch(() => ({}));
      if (!response.ok) {
        throw new Error(data.error || "Failed to delete team");
      }
      setStatusType("success");
      setStatusMessage("Team deleted");
      setSelectedTeamId("");
      setTeamDetail(null);
      await loadOwnerData();
      setTab("teams");
    } catch (error) {
      setStatusType("danger");
      setStatusMessage(error.message || "Failed to delete team");
    }
  }

  async function handleApproval(approvalId, action) {
    const route = action === "approve"
      ? fillRoute(ROUTE_TEMPLATES.pendingApprovalApprove, { id: approvalId })
      : fillRoute(ROUTE_TEMPLATES.pendingApprovalReject, { id: approvalId });
    try {
      const response = await authClient.apiFetch(route, {
        method: "PUT"
      });
      const data = await response.json().catch(() => ({}));
      if (!response.ok) {
        throw new Error(data.error || `Failed to ${action} approval`);
      }
      setStatusType("success");
      setStatusMessage(data.message || `Approval ${action}d`);
      await loadTeamContext(selectedTeamId);
      await loadOwnerData();
    } catch (error) {
      setStatusType("danger");
      setStatusMessage(error.message || `Failed to ${action} approval`);
    }
  }

  async function createOwnerInviteLink(event) {
    event.preventDefault();
    if (!selectedTeamId) return;
    const payload = {
      teamId: selectedTeamId,
      expiresIn: linkExpiresIn
    };
    if (Number(linkMaxUses) > 0) {
      payload.maxUses = Number(linkMaxUses);
    }

    try {
      const response = await authClient.apiFetch("/api/owner/invite-links", {
        method: "POST",
        body: JSON.stringify(payload)
      });
      const data = await response.json().catch(() => ({}));
      if (!response.ok) {
        throw new Error(data.error || "Failed to create invite link");
      }
      setStatusType("success");
      setStatusMessage("Invite link created");
      setLinkMaxUses("");
      await loadTeamContext(selectedTeamId);
    } catch (error) {
      setStatusType("danger");
      setStatusMessage(error.message || "Failed to create invite link");
    }
  }

  async function deleteInviteLink(linkId) {
    try {
      const response = await authClient.apiFetch(`/api/owner/invite-links/${linkId}`, {
        method: "DELETE"
      });
      const data = await response.json().catch(() => ({}));
      if (!response.ok) {
        throw new Error(data.error || "Failed to delete invite link");
      }
      setStatusType("success");
      setStatusMessage("Invite link deleted");
      await loadTeamContext(selectedTeamId);
    } catch (error) {
      setStatusType("danger");
      setStatusMessage(error.message || "Failed to delete invite link");
    }
  }

  async function createTeamInvite(event) {
    event.preventDefault();
    if (!inviteTeamId || !invitePlayerIdentifier.trim()) return;
    try {
      const response = await authClient.apiFetch(`/api/teams/${inviteTeamId}/invite`, {
        method: "POST",
        body: JSON.stringify({ playerIdentifier: invitePlayerIdentifier.trim() })
      });
      const data = await response.json().catch(() => ({}));
      if (!response.ok) throw new Error(data.error || "Failed to create team invite");
      setStatusType("success");
      setStatusMessage("Team invite created");
      setInvitePlayerIdentifier("");
      await loadTeamInvites(inviteTeamId);
    } catch (error) {
      setStatusType("danger");
      setStatusMessage(error.message || "Failed to create team invite");
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
      setStatusType("danger");
      setStatusMessage(error.message || "Failed to load team invites");
    }
  }

  async function loadOwnerTournamentRequests() {
    try {
      const response = await authClient.apiFetch("/api/owner/tournament-requests");
      const data = await response.json().catch(() => []);
      if (!response.ok) throw new Error(data.error || "Failed to load tournament requests");
      setTournamentRequests(safeArray(data));
    } catch (error) {
      setStatusType("danger");
      setStatusMessage(error.message || "Failed to load tournament requests");
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
      setStatusType("danger");
      setStatusMessage(error.message || "Failed to lookup team by alias route");
    }
  }

  async function submitRaidResult(event) {
    event.preventDefault();
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
      setStatusType("success");
      setStatusMessage("Raid processed successfully");
    } catch (error) {
      setStatusType("danger");
      setStatusMessage(error.message || "Failed to process raid");
    }
  }

  async function generateLegacyTeamLink() {
    if (!selectedTeamId) return;
    try {
      const response = await authClient.apiFetch(`/api/teams/${selectedTeamId}/generate-link`, { method: "POST" });
      const data = await response.json().catch(() => ({}));
      if (!response.ok) throw new Error(data.error || "Failed to generate team link");
      setGeneratedLink(data.invite_url || `${window.location.origin}/invite/team/${data.invite_token || ""}`);
      setStatusType("success");
      setStatusMessage("Team invite link generated");
    } catch (error) {
      setStatusType("danger");
      setStatusMessage(error.message || "Failed to generate team link");
    }
  }

  const selectedInvites = invites[inviteTab] || [];
  const teamPlayers = safeArray(teamDetail?.Players || teamDetail?.players);

  return (
    <section className="panel-card">
      {statusMessage ? <div className={`status-box ${statusType}`}>{statusMessage}</div> : null}
      <div className="tab-row">
        {renderTab(tab, setTab, "teams", `My Teams (${teams.length})`)}
        {renderTab(tab, setTab, "invites", "Event Invitations")}
        {renderTab(tab, setTab, "manage", "Team Manager")}
        {renderTab(tab, setTab, "approvals", `Pending Approvals (${pendingApprovals.length})`)}
        {renderTab(tab, setTab, "links", `Invite Links (${inviteLinks.length})`)}
        {renderTab(tab, setTab, "advanced", "Advanced APIs")}
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
              <div className="card-actions">
                <button
                  className="btn-outline"
                  onClick={() => {
                    setSelectedTeamId(getTeamId(team));
                    setTab("manage");
                  }}
                >
                  Manage Team
                </button>
              </div>
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

      {tab === "manage" ? (
        <div className="list-wrap">
          <label>Select Team</label>
          <select className="inline-select" value={selectedTeamId} onChange={(event) => setSelectedTeamId(event.target.value)}>
            <option value="">Select a team...</option>
            {teams.map((team) => (
              <option key={getTeamId(team)} value={getTeamId(team)}>{team.TeamName || team.teamName || "Team"}</option>
            ))}
          </select>

          {!selectedTeamId ? <div className="empty-state">Choose a team to manage.</div> : null}
          {selectedTeamId && teamDetail ? (
            <>
              <article className="entity-card">
                <h4>Update Team</h4>
                <form className="form-grid" onSubmit={updateTeamDetails}>
                  <label>Team Name</label>
                  <input value={teamEditName} onChange={(event) => setTeamEditName(event.target.value)} />
                  <label>City</label>
                  <input value={teamEditCity} onChange={(event) => setTeamEditCity(event.target.value)} placeholder="Optional" />
                  <button className="btn-primary-orange" type="submit">Save Team</button>
                </form>
              </article>

              <article className="entity-card">
                <h4>Add Player</h4>
                <form className="form-grid" onSubmit={addPlayerToSelectedTeam}>
                  <label>Player Username or Email</label>
                  <input value={playerIdentifier} onChange={(event) => setPlayerIdentifier(event.target.value)} placeholder="player username/email" />
                  <button className="btn-primary-orange" type="submit">Add Player</button>
                </form>
              </article>

              <article className="entity-card">
                <h4>Team Players ({teamPlayers.length})</h4>
                {!teamPlayers.length ? <p>No players in this team.</p> : null}
                {teamPlayers.map((player) => (
                  <div className="card-actions" key={player._id || player.id || player.userId}>
                    <span>{player.fullName || player.userId || player._id}</span>
                    <button className="btn-outline" onClick={() => removePlayerFromSelectedTeam(player._id)}>Remove</button>
                  </div>
                ))}
              </article>

              <article className="entity-card">
                <h4>Danger Zone</h4>
                <p>Delete this team permanently.</p>
                <button className="btn-outline" onClick={deleteSelectedTeam}>Delete Team</button>
              </article>
            </>
          ) : null}
        </div>
      ) : null}

      {tab === "approvals" ? (
        <div className="list-wrap">
          <label>Select Team</label>
          <select className="inline-select" value={selectedTeamId} onChange={(event) => setSelectedTeamId(event.target.value)}>
            <option value="">Select a team...</option>
            {teams.map((team) => (
              <option key={getTeamId(team)} value={getTeamId(team)}>{team.TeamName || team.teamName || "Team"}</option>
            ))}
          </select>

          {!selectedTeamId ? <div className="empty-state">Choose a team to view pending approvals.</div> : null}
          {selectedTeamId && !pendingApprovals.length ? <div className="empty-state">No pending approvals for this team.</div> : null}
          {pendingApprovals.map((approval) => (
            <article className="request-card" key={approval.ID || approval._id}>
              <h4>{approval.AcceptorName || approval.acceptorName || "Pending Request"}</h4>
              <p>Username: {approval.AcceptorUsername || approval.acceptorUsername || "-"}</p>
              <p>Role: {approval.AcceptorRole || approval.acceptorRole || "-"}</p>
              <p>Status: <span className="badge-custom">{approval.Status || approval.status || "pending"}</span></p>
              <div className="card-actions">
                <button className="btn-primary-orange" onClick={() => handleApproval(approval.ID || approval._id, "approve")}>Approve</button>
                <button className="btn-outline" onClick={() => handleApproval(approval.ID || approval._id, "reject")}>Reject</button>
              </div>
            </article>
          ))}
        </div>
      ) : null}

      {tab === "links" ? (
        <div className="list-wrap">
          <label>Select Team</label>
          <select className="inline-select" value={selectedTeamId} onChange={(event) => setSelectedTeamId(event.target.value)}>
            <option value="">Select a team...</option>
            {teams.map((team) => (
              <option key={getTeamId(team)} value={getTeamId(team)}>{team.TeamName || team.teamName || "Team"}</option>
            ))}
          </select>

          {selectedTeamId ? (
            <article className="entity-card">
              <h4>Create Invite Link</h4>
              <form className="form-grid" onSubmit={createOwnerInviteLink}>
                <label>Expires In</label>
                <select value={linkExpiresIn} onChange={(event) => setLinkExpiresIn(event.target.value)}>
                  <option value="1h">1 hour</option>
                  <option value="24h">24 hours</option>
                  <option value="7d">7 days</option>
                  <option value="30d">30 days</option>
                  <option value="never">Never</option>
                </select>
                <label>Max Uses (optional)</label>
                <input type="number" min={1} value={linkMaxUses} onChange={(event) => setLinkMaxUses(event.target.value)} placeholder="Unlimited if empty" />
                <button className="btn-primary-orange" type="submit">Create Link</button>
              </form>
            </article>
          ) : (
            <div className="empty-state">Choose a team to manage invite links.</div>
          )}

          {selectedTeamId && !inviteLinks.length ? <div className="empty-state">No active invite links for this team.</div> : null}
          {inviteLinks.map((link) => (
            <article className="request-card" key={link.id}>
              <h4>{link.targetName || "Team Invite Link"}</h4>
              <p>Token: {link.code}</p>
              <p>Link: <a href={`/invite/team/${link.code}`} target="_blank" rel="noreferrer">/invite/team/{link.code}</a></p>
              <p>Uses: {link.usesCount || 0}{link.maxUses ? ` / ${link.maxUses}` : ""}</p>
              <div className="card-actions">
                <button className="btn-outline" onClick={() => deleteInviteLink(link.id)}>Deactivate</button>
              </div>
            </article>
          ))}
        </div>
      ) : null}

      {tab === "advanced" ? (
        <div className="list-wrap">
          <article className="entity-card">
            <h4>Create Team Invite (`/api/teams/:id/invite`)</h4>
            <form className="form-grid" onSubmit={createTeamInvite}>
              <label>Team</label>
              <select value={inviteTeamId} onChange={(event) => setInviteTeamId(event.target.value)}>
                <option value="">Select team...</option>
                {teams.map((team) => (
                  <option key={getTeamId(team)} value={getTeamId(team)}>{team.TeamName || team.teamName || "Team"}</option>
                ))}
              </select>
              <label>Player Username/Email</label>
              <input value={invitePlayerIdentifier} onChange={(event) => setInvitePlayerIdentifier(event.target.value)} placeholder="player identifier" />
              <button className="btn-primary-orange" type="submit">Send Team Invite</button>
              <button className="btn-outline" type="button" onClick={generateLegacyTeamLink}>Generate Legacy Team Link</button>
            </form>
            {generatedLink ? <p>Generated Link: <a href={generatedLink} target="_blank" rel="noreferrer">{generatedLink}</a></p> : null}
          </article>

          <article className="entity-card">
            <h4>Team Invites (`/api/teams/:id/invites`)</h4>
            <div className="card-actions">
              <button className="btn-outline" onClick={() => loadTeamInvites()}>Load Team Invites</button>
            </div>
            {!teamInvites.length ? <p>No team invites loaded.</p> : null}
            {teamInvites.map((invite) => (
              <p key={getInviteId(invite)}>{invite.toName || invite.playerName || invite.toUsername || "Invite"} — {normalizeStatus(invite)}</p>
            ))}
          </article>

          <article className="entity-card">
            <h4>Tournament Requests (`/api/owner/tournament-requests`)</h4>
            <div className="card-actions">
              <button className="btn-outline" onClick={loadOwnerTournamentRequests}>Load Requests</button>
            </div>
            {!tournamentRequests.length ? <p>No tournament requests loaded.</p> : null}
            {tournamentRequests.map((req, index) => (
              <p key={req.id || req._id || index}>{req.eventName || req.tournamentName || "Request"} — {req.status || "pending"}</p>
            ))}
          </article>

          <article className="entity-card">
            <h4>Team Alias Lookup (`/api/team/:id`)</h4>
            <div className="form-grid">
              <label>Team ID</label>
              <input value={lookupTeamId} onChange={(event) => setLookupTeamId(event.target.value)} placeholder="team id" />
              <button className="btn-outline" onClick={lookupTeamAlias}>Lookup Team</button>
            </div>
            {lookupTeamData ? <pre style={{ whiteSpace: "pre-wrap" }}>{JSON.stringify(lookupTeamData, null, 2)}</pre> : null}
          </article>

          <article className="entity-card">
            <h4>Matches + Endgame Routes</h4>
            <div className="card-actions">
              <button className="btn-outline" onClick={() => window.open("/api/matches", "_blank")}>Open `/api/matches`</button>
              <button className="btn-outline" onClick={() => window.open("/api/endgame", "_blank")}>Open `/api/endgame`</button>
              <button className="btn-outline" onClick={() => window.open("/endgame", "_blank")}>Open `/endgame`</button>
            </div>
            <div className="form-grid">
              <label>Match ID (`/api/matches/:id`)</label>
              <input value={matchLookupId} onChange={(event) => setMatchLookupId(event.target.value)} placeholder="match id" />
              <button className="btn-outline" onClick={() => matchLookupId && window.open(`/api/matches/${matchLookupId}`, "_blank")}>Open Match Route</button>
            </div>
          </article>

          <article className="entity-card">
            <h4>Submit Raid (`/api/matches/raid`)</h4>
            <form className="form-grid" onSubmit={submitRaidResult}>
              <label>Raid Type</label>
              <select value={raidType} onChange={(event) => setRaidType(event.target.value)}>
                <option value="successful">successful</option>
                <option value="defense">defense</option>
                <option value="empty">empty</option>
              </select>
              <label>Raider ID</label>
              <input value={raiderId} onChange={(event) => setRaiderId(event.target.value)} placeholder="raider id" />
              <label>Defender IDs (comma separated)</label>
              <input value={defenderIds} onChange={(event) => setDefenderIds(event.target.value)} placeholder="def1,def2" />
              <label><input type="checkbox" checked={bonusTaken} onChange={(event) => setBonusTaken(event.target.checked)} /> Bonus Taken</label>
              <button className="btn-primary-orange" type="submit">Submit Raid</button>
            </form>
          </article>
        </div>
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
  const [selectedEventId, setSelectedEventId] = useState("");
  const [eventDetail, setEventDetail] = useState(null);
  const [eventEditName, setEventEditName] = useState("");
  const [eventEditType, setEventEditType] = useState("match");
  const [eventEditMaxTeams, setEventEditMaxTeams] = useState(4);
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
  const [inviteOwnerIdentifier, setInviteOwnerIdentifier] = useState("");
  const [linkExpiresIn, setLinkExpiresIn] = useState("30d");
  const [linkMaxUses, setLinkMaxUses] = useState("");
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

      if (!selectedEventId && eventData.length > 0) {
        setSelectedEventId(eventData[0].id || "");
      }
    } catch (error) {
      setStatusType("danger");
      setStatusMessage(error.message || "Failed to load organizer dashboard data");
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

      if (!eventRes.ok) {
        throw new Error(eventData.error || "Failed to load event details");
      }
      if (!approvalsRes.ok) {
        throw new Error(approvalsData.error || "Failed to load pending approvals");
      }
      if (!linksRes.ok) {
        throw new Error(linksData.error || "Failed to load invite links");
      }

      setEventDetail(eventData);
      setEventEditName(eventData.eventName || "");
      setEventEditType(eventData.eventType || "match");
      setEventEditMaxTeams(eventData.maxTeams || 4);
      setPendingApprovals(safeArray(approvalsData));
      setInviteLinks(safeArray(linksData.data).filter((link) => (link.targetId || "") === eventId));
      await loadEventOpsData(eventId, eventData.eventType || "match");
    } catch (error) {
      setStatusType("danger");
      setStatusMessage(error.message || "Failed to load event context");
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

      if (teamsRes.ok) {
        setEventTeams(safeArray(teamsData));
      }
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
        if (fixturesRes.ok) {
          setTournamentFixtures(safeArray(fixturesData.fixtures));
        }
        if (standingsRes.ok) {
          setTournamentStandings(safeArray(standingsData.standings));
        }
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
        if (champRes.ok) {
          setChampionshipInfo(champData);
        }
        if (champFixturesRes.ok) {
          setChampionshipFixtures(safeArray(champFixturesData));
        }
        if (champStatsRes.ok) {
          setChampionshipStats(safeArray(champStatsData));
        }
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
    loadOrganizerData();
  }, []);

  useEffect(() => {
    if (!selectedEventId) return;
    loadEventContext(selectedEventId);
  }, [selectedEventId]);

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

  async function updateEventDetails(event) {
    event.preventDefault();
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
      if (!response.ok) {
        throw new Error(data.error || "Failed to update event");
      }
      setStatusType("success");
      setStatusMessage("Event updated");
      await loadOrganizerData();
      await loadEventContext(selectedEventId);
    } catch (error) {
      setStatusType("danger");
      setStatusMessage(error.message || "Failed to update event");
    }
  }

  async function markEventCompleted(eventId) {
    if (!eventId) return;
    try {
      const response = await authClient.apiFetch(`/api/events/${eventId}/complete`, {
        method: "POST"
      });
      const data = await response.json().catch(() => ({}));
      if (!response.ok) {
        throw new Error(data.error || "Failed to mark event completed");
      }
      setStatusType("success");
      setStatusMessage("Event marked completed");
      await loadOrganizerData();
      await loadEventContext(eventId);
    } catch (error) {
      setStatusType("danger");
      setStatusMessage(error.message || "Failed to complete event");
    }
  }

  async function startEvent(eventId) {
    if (!eventId) return;
    try {
      const response = await authClient.apiFetch(`/api/organizer/events/${eventId}/start`, {
        method: "POST"
      });
      const data = await response.json().catch(() => ({}));
      if (!response.ok) {
        throw new Error(data.error || "Failed to start event");
      }
      setStatusType("success");
      setStatusMessage("Event started");
      await loadOrganizerData();
      await loadEventContext(eventId);
    } catch (error) {
      setStatusType("danger");
      setStatusMessage(error.message || "Failed to start event");
    }
  }

  async function sendEventInvite(event) {
    event.preventDefault();
    if (!selectedEventId || !inviteOwnerIdentifier.trim()) return;

    try {
      const response = await authClient.apiFetch(`/api/events/${selectedEventId}/invite`, {
        method: "POST",
        body: JSON.stringify({ ownerIdentifier: inviteOwnerIdentifier.trim() })
      });
      const data = await response.json().catch(() => ({}));
      if (!response.ok) {
        throw new Error(data.error || "Failed to send invite");
      }
      setInviteOwnerIdentifier("");
      setStatusType("success");
      setStatusMessage("Invitation created");
      await loadOrganizerData();
      await loadEventContext(selectedEventId);
    } catch (error) {
      setStatusType("danger");
      setStatusMessage(error.message || "Failed to send invite");
    }
  }

  async function handleApproval(approvalId, action) {
    const route = action === "approve"
      ? fillRoute(ROUTE_TEMPLATES.pendingApprovalApprove, { id: approvalId })
      : fillRoute(ROUTE_TEMPLATES.pendingApprovalReject, { id: approvalId });
    try {
      const response = await authClient.apiFetch(route, {
        method: "PUT"
      });
      const data = await response.json().catch(() => ({}));
      if (!response.ok) {
        throw new Error(data.error || `Failed to ${action} approval`);
      }
      setStatusType("success");
      setStatusMessage(data.message || `Approval ${action}d`);
      await loadEventContext(selectedEventId);
      await loadOrganizerData();
    } catch (error) {
      setStatusType("danger");
      setStatusMessage(error.message || `Failed to ${action} approval`);
    }
  }

  async function createOrganizerInviteLink(event) {
    event.preventDefault();
    if (!selectedEventId) return;

    const payload = {
      targetId: selectedEventId,
      expiresIn: linkExpiresIn
    };
    if (Number(linkMaxUses) > 0) {
      payload.maxUses = Number(linkMaxUses);
    }

    try {
      const response = await authClient.apiFetch("/api/organizer/invite-links", {
        method: "POST",
        body: JSON.stringify(payload)
      });
      const data = await response.json().catch(() => ({}));
      if (!response.ok) {
        throw new Error(data.error || "Failed to create invite link");
      }
      setLinkMaxUses("");
      setStatusType("success");
      setStatusMessage("Invite link created");
      await loadEventContext(selectedEventId);
    } catch (error) {
      setStatusType("danger");
      setStatusMessage(error.message || "Failed to create invite link");
    }
  }

  async function deleteInviteLink(linkId) {
    try {
      const response = await authClient.apiFetch(`/api/organizer/invite-links/${linkId}`, {
        method: "DELETE"
      });
      const data = await response.json().catch(() => ({}));
      if (!response.ok) {
        throw new Error(data.error || "Failed to delete invite link");
      }
      setStatusType("success");
      setStatusMessage("Invite link deleted");
      await loadEventContext(selectedEventId);
    } catch (error) {
      setStatusType("danger");
      setStatusMessage(error.message || "Failed to delete invite link");
    }
  }

  async function initializeTournament(eventId) {
    if (!eventId) return;
    try {
      const response = await authClient.apiFetch(`/api/tournaments/initialize/${eventId}`, {
        method: "POST"
      });
      const data = await response.json().catch(() => ({}));
      if (!response.ok) {
        throw new Error(data.error || "Failed to initialize tournament");
      }
      setStatusType("success");
      setStatusMessage(data.message || "Tournament initialized");
      await loadEventContext(eventId);
    } catch (error) {
      setStatusType("danger");
      setStatusMessage(error.message || "Failed to initialize tournament");
    }
  }

  async function startTournamentFixture(fixtureId) {
    if (!selectedEventId || !fixtureId) return;
    try {
      const response = await authClient.apiFetch(`/api/tournaments/${selectedEventId}/start-match/${fixtureId}`, {
        method: "POST"
      });
      const data = await response.json().catch(() => ({}));
      if (!response.ok) {
        throw new Error(data.error || "Failed to start fixture match");
      }
      setStatusType("success");
      setStatusMessage(`Tournament match started: ${data.matchId || "created"}`);
      await loadEventContext(selectedEventId);
    } catch (error) {
      setStatusType("danger");
      setStatusMessage(error.message || "Failed to start fixture match");
    }
  }

  async function initializeChampionship(eventId) {
    if (!eventId) return;
    try {
      const response = await authClient.apiFetch(`/api/championships/initialize/${eventId}`, {
        method: "POST"
      });
      const data = await response.json().catch(() => ({}));
      if (!response.ok) {
        throw new Error(data.error || "Failed to initialize championship");
      }
      setStatusType("success");
      setStatusMessage(data.message || "Championship initialized");
      await loadEventContext(eventId);
    } catch (error) {
      setStatusType("danger");
      setStatusMessage(error.message || "Failed to initialize championship");
    }
  }

  async function startChampionshipFixture(fixtureId) {
    if (!fixtureId) return;
    const championshipId = getObjectIdString(championshipInfo?.id || championshipInfo?._id);
    if (!championshipId) {
      setStatusType("warning");
      setStatusMessage("Championship ID not found");
      return;
    }

    try {
      const response = await authClient.apiFetch(`/api/championships/${championshipId}/start-match/${fixtureId}`, {
        method: "POST"
      });
      const data = await response.json().catch(() => ({}));
      if (!response.ok) {
        throw new Error(data.error || "Failed to start championship match");
      }
      setStatusType("success");
      setStatusMessage(`Championship match started: ${data.matchId || "created"}`);
      await loadEventContext(selectedEventId);
    } catch (error) {
      setStatusType("danger");
      setStatusMessage(error.message || "Failed to start championship match");
    }
  }

  async function submitRaidResult(event) {
    event.preventDefault();
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
      setStatusType("success");
      setStatusMessage("Raid processed successfully");
    } catch (error) {
      setStatusType("danger");
      setStatusMessage(error.message || "Failed to process raid");
    }
  }

  async function generateLegacyEventLink() {
    if (!selectedEventId) return;
    try {
      const response = await authClient.apiFetch(`/api/events/${selectedEventId}/generate-link`, { method: "POST" });
      const data = await response.json().catch(() => ({}));
      if (!response.ok) throw new Error(data.error || "Failed to generate event link");
      setGeneratedEventLink(data.invite_url || `${window.location.origin}/invite/event/${data.invite_token || ""}`);
      setStatusType("success");
      setStatusMessage("Event invite link generated");
    } catch (error) {
      setStatusType("danger");
      setStatusMessage(error.message || "Failed to generate event link");
    }
  }

  return (
    <section className="panel-card">
      {statusMessage ? <div className={`status-box ${statusType}`}>{statusMessage}</div> : null}
      <div className="tab-row">
        {renderTab(tab, setTab, "events", "My Events")}
        {renderTab(tab, setTab, "invites", "Invite Responses")}
        {renderTab(tab, setTab, "manage", "Event Manager")}
        {renderTab(tab, setTab, "competition", "Tournament/Championship")}
        {renderTab(tab, setTab, "advanced", "Advanced APIs")}
        {renderTab(tab, setTab, "approvals", `Pending Approvals (${pendingApprovals.length})`)}
        {renderTab(tab, setTab, "links", `Invite Links (${inviteLinks.length})`)}
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
              <article
                key={event.id}
                className="entity-card clickable"
                onClick={() => {
                  setSelectedEventId(event.id);
                  setTab("manage");
                }}
              >
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

      {tab === "manage" ? (
        <div className="list-wrap">
          <label>Select Event</label>
          <select className="inline-select" value={selectedEventId} onChange={(event) => setSelectedEventId(event.target.value)}>
            <option value="">Select an event...</option>
            {events.map((event) => (
              <option key={event.id} value={event.id}>{event.eventName || "Event"}</option>
            ))}
          </select>

          {!selectedEventId ? <div className="empty-state">Choose an event to manage.</div> : null}
          {selectedEventId && eventDetail ? (
            <>
              <article className="entity-card">
                <h4>Update Event</h4>
                <form className="form-grid" onSubmit={updateEventDetails}>
                  <label>Event Name</label>
                  <input value={eventEditName} onChange={(event) => setEventEditName(event.target.value)} />
                  <label>Event Type</label>
                  <select value={eventEditType} onChange={(event) => setEventEditType(event.target.value)}>
                    <option value="match">Match</option>
                    <option value="tournament">Tournament</option>
                    <option value="championship">Championship</option>
                  </select>
                  {eventEditType !== "match" ? (
                    <>
                      <label>No. of Teams</label>
                      <input type="number" min={4} value={eventEditMaxTeams} onChange={(event) => setEventEditMaxTeams(event.target.value)} />
                    </>
                  ) : null}
                  <button className="btn-primary-orange" type="submit">Save Event</button>
                </form>
              </article>

              <article className="entity-card">
                <h4>Invite Team Owner</h4>
                <form className="form-grid" onSubmit={sendEventInvite}>
                  <label>Owner Username or Email</label>
                  <input value={inviteOwnerIdentifier} onChange={(event) => setInviteOwnerIdentifier(event.target.value)} placeholder="owner identifier" />
                  <button className="btn-primary-orange" type="submit">Send Invite</button>
                </form>
              </article>

              <article className="entity-card">
                <h4>Event Actions</h4>
                <p>Status: <span className="badge-custom">{eventDetail.status || "-"}</span></p>
                <div className="card-actions">
                  <button className="btn-primary-orange" onClick={() => startEvent(selectedEventId)}>Start Event</button>
                  <button className="btn-outline" onClick={() => markEventCompleted(selectedEventId)}>Mark Completed</button>
                  <button className="btn-outline" onClick={() => window.location.assign(`/organizer/event/${selectedEventId}`)}>Open Event Page</button>
                </div>
              </article>

              <article className="entity-card">
                <h4>Participating Teams ({eventTeams.length})</h4>
                {!eventTeams.length ? <p>No accepted teams yet.</p> : null}
                {eventTeams.map((team) => (
                  <p key={team.teamId || team.TeamID || team.ID || team.teamName}>{team.teamName || team.TeamName || team.name || "Team"} — {team.status || team.Status || "accepted"}</p>
                ))}
              </article>

              <article className="entity-card">
                <h4>Latest Match Stats</h4>
                {!eventMatchStats ? <p>No match stats found yet.</p> : null}
                {eventMatchStats ? (
                  <>
                    <p>Match ID: {eventMatchStats.matchId || eventMatchStats._id || "-"}</p>
                    <p>Team A: {eventMatchStats?.data?.TeamAScore ?? "-"} | Team B: {eventMatchStats?.data?.TeamBScore ?? "-"}</p>
                  </>
                ) : null}
              </article>
            </>
          ) : null}
        </div>
      ) : null}

      {tab === "competition" ? (
        <div className="list-wrap">
          <label>Select Event</label>
          <select className="inline-select" value={selectedEventId} onChange={(event) => setSelectedEventId(event.target.value)}>
            <option value="">Select an event...</option>
            {events.map((event) => (
              <option key={event.id} value={event.id}>{event.eventName || "Event"}</option>
            ))}
          </select>

          {!selectedEventId || !eventDetail ? <div className="empty-state">Choose an event to manage competition data.</div> : null}

          {selectedEventId && eventDetail?.eventType === "tournament" ? (
            <>
              <article className="entity-card">
                <h4>Tournament Actions</h4>
                <div className="card-actions">
                  <button className="btn-primary-orange" onClick={() => initializeTournament(selectedEventId)}>Initialize Tournament</button>
                </div>
              </article>

              <article className="entity-card">
                <h4>Fixtures ({tournamentFixtures.length})</h4>
                {!tournamentFixtures.length ? <p>No fixtures found.</p> : null}
                {tournamentFixtures.map((fixture) => (
                  <div key={fixture.id || fixture._id} className="request-card">
                    <p>{fixture.team1Name || "Team A"} vs {fixture.team2Name || "Team B"}</p>
                    <p>Status: <span className="badge-custom">{fixture.status || "pending"}</span></p>
                    {(fixture.status || "").toLowerCase() === "pending" ? (
                      <button className="btn-outline" onClick={() => startTournamentFixture(fixture.id || fixture._id)}>Start Match</button>
                    ) : null}
                  </div>
                ))}
              </article>

              <article className="entity-card">
                <h4>Standings ({tournamentStandings.length})</h4>
                {!tournamentStandings.length ? <p>No standings yet.</p> : null}
                {tournamentStandings.map((item) => (
                  <p key={`${item.teamId}-${item.position}`}>#{item.position} {item.teamName} — {item.points} pts (NRR {item.nrr})</p>
                ))}
              </article>
            </>
          ) : null}

          {selectedEventId && eventDetail?.eventType === "championship" ? (
            <>
              <article className="entity-card">
                <h4>Championship Actions</h4>
                <div className="card-actions">
                  <button className="btn-primary-orange" onClick={() => initializeChampionship(selectedEventId)}>Initialize Championship</button>
                </div>
              </article>

              <article className="entity-card">
                <h4>Championship Details</h4>
                {!championshipInfo ? <p>Not initialized yet.</p> : null}
                {championshipInfo ? (
                  <>
                    <p>ID: {getObjectIdString(championshipInfo.id || championshipInfo._id) || "-"}</p>
                    <p>Status: <span className="badge-custom">{championshipInfo.status || "-"}</span></p>
                    <p>Round: {championshipInfo.currentRound || "-"} / {championshipInfo.totalRounds || "-"}</p>
                  </>
                ) : null}
              </article>

              <article className="entity-card">
                <h4>Fixtures ({championshipFixtures.length})</h4>
                {!championshipFixtures.length ? <p>No fixtures found.</p> : null}
                {championshipFixtures.map((fixture) => {
                  const fixtureId = getObjectIdString(fixture.id || fixture._id);
                  const isPending = String(fixture.status || "").toLowerCase() === "pending";
                  return (
                    <div key={fixtureId || `${fixture.roundNumber || "r"}-${getObjectIdString(fixture.team1Id) || "t1"}-${getObjectIdString(fixture.team2Id) || "bye"}`} className="request-card">
                      <p>{fixture.team1?.name || fixture.team1?.teamName || "Team A"} vs {fixture.team2?.name || fixture.team2?.teamName || (fixture.isBye ? "BYE" : "Team B")}</p>
                      <p>Status: <span className="badge-custom">{fixture.status || "pending"}</span></p>
                      {isPending && fixtureId ? (
                        <button className="btn-outline" onClick={() => startChampionshipFixture(fixtureId)}>Start Match</button>
                      ) : null}
                    </div>
                  );
                })}
              </article>

              <article className="entity-card">
                <h4>Stats ({championshipStats.length})</h4>
                {!championshipStats.length ? <p>No stats yet.</p> : null}
                {championshipStats.map((item) => (
                  <p key={getObjectIdString(item.id || item._id) || item.team?.name}>{item.team?.name || item.team?.teamName || "Team"} — NRR {item.nrr || 0}</p>
                ))}
              </article>
            </>
          ) : null}
        </div>
      ) : null}

      {tab === "advanced" ? (
        <div className="list-wrap">
          <article className="entity-card">
            <h4>Match Routes</h4>
            <div className="card-actions">
              <button className="btn-outline" onClick={() => window.open("/api/matches", "_blank")}>Open `/api/matches`</button>
              <button className="btn-outline" onClick={() => window.open("/api/endgame", "_blank")}>Open `/api/endgame`</button>
              <button className="btn-outline" onClick={() => window.open("/endgame", "_blank")}>Open `/endgame`</button>
              <button className="btn-outline" onClick={() => window.open("/scorer", "_blank")}>Open `/scorer`</button>
              <button className="btn-outline" onClick={generateLegacyEventLink}>Generate Legacy Event Link</button>
            </div>
            {generatedEventLink ? <p>Generated Link: <a href={generatedEventLink} target="_blank" rel="noreferrer">{generatedEventLink}</a></p> : null}
            <div className="form-grid">
              <label>Match ID (`/api/matches/:id`)</label>
              <input value={matchLookupId} onChange={(event) => setMatchLookupId(event.target.value)} placeholder="match id" />
              <button className="btn-outline" onClick={() => matchLookupId && window.open(`/api/matches/${matchLookupId}`, "_blank")}>Open Match Route</button>
              <label>Player Selection Match ID</label>
              <input value={playerSelectionMatchId} onChange={(event) => setPlayerSelectionMatchId(event.target.value)} placeholder="match id" />
              <button className="btn-outline" onClick={() => playerSelectionMatchId && window.open(`/organizer/playerselection/${playerSelectionMatchId}`, "_blank")}>Open Player Selection</button>
            </div>
          </article>

          <article className="entity-card">
            <h4>Submit Raid (`/api/matches/raid`)</h4>
            <form className="form-grid" onSubmit={submitRaidResult}>
              <label>Raid Type</label>
              <select value={raidType} onChange={(event) => setRaidType(event.target.value)}>
                <option value="successful">successful</option>
                <option value="defense">defense</option>
                <option value="empty">empty</option>
              </select>
              <label>Raider ID</label>
              <input value={raiderId} onChange={(event) => setRaiderId(event.target.value)} placeholder="raider id" />
              <label>Defender IDs (comma separated)</label>
              <input value={defenderIds} onChange={(event) => setDefenderIds(event.target.value)} placeholder="def1,def2" />
              <label><input type="checkbox" checked={bonusTaken} onChange={(event) => setBonusTaken(event.target.checked)} /> Bonus Taken</label>
              <button className="btn-primary-orange" type="submit">Submit Raid</button>
            </form>
          </article>
        </div>
      ) : null}

      {tab === "approvals" ? (
        <div className="list-wrap">
          <label>Select Event</label>
          <select className="inline-select" value={selectedEventId} onChange={(event) => setSelectedEventId(event.target.value)}>
            <option value="">Select an event...</option>
            {events.map((event) => (
              <option key={event.id} value={event.id}>{event.eventName || "Event"}</option>
            ))}
          </select>

          {!selectedEventId ? <div className="empty-state">Choose an event to view approvals.</div> : null}
          {selectedEventId && !pendingApprovals.length ? <div className="empty-state">No pending approvals for this event.</div> : null}
          {pendingApprovals.map((approval) => (
            <article className="request-card" key={approval.ID || approval._id}>
              <h4>{approval.AcceptorName || approval.acceptorName || "Pending Request"}</h4>
              <p>Username: {approval.AcceptorUsername || approval.acceptorUsername || "-"}</p>
              <p>Role: {approval.AcceptorRole || approval.acceptorRole || "-"}</p>
              <p>Status: <span className="badge-custom">{approval.Status || approval.status || "pending"}</span></p>
              <div className="card-actions">
                <button className="btn-primary-orange" onClick={() => handleApproval(approval.ID || approval._id, "approve")}>Approve</button>
                <button className="btn-outline" onClick={() => handleApproval(approval.ID || approval._id, "reject")}>Reject</button>
              </div>
            </article>
          ))}
        </div>
      ) : null}

      {tab === "links" ? (
        <div className="list-wrap">
          <label>Select Event</label>
          <select className="inline-select" value={selectedEventId} onChange={(event) => setSelectedEventId(event.target.value)}>
            <option value="">Select an event...</option>
            {events.map((event) => (
              <option key={event.id} value={event.id}>{event.eventName || "Event"}</option>
            ))}
          </select>

          {selectedEventId ? (
            <article className="entity-card">
              <h4>Create Invite Link</h4>
              <form className="form-grid" onSubmit={createOrganizerInviteLink}>
                <label>Expires In</label>
                <select value={linkExpiresIn} onChange={(event) => setLinkExpiresIn(event.target.value)}>
                  <option value="1h">1 hour</option>
                  <option value="24h">24 hours</option>
                  <option value="7d">7 days</option>
                  <option value="30d">30 days</option>
                  <option value="never">Never</option>
                </select>
                <label>Max Uses (optional)</label>
                <input type="number" min={1} value={linkMaxUses} onChange={(event) => setLinkMaxUses(event.target.value)} placeholder="Unlimited if empty" />
                <button className="btn-primary-orange" type="submit">Create Link</button>
              </form>
            </article>
          ) : (
            <div className="empty-state">Choose an event to manage invite links.</div>
          )}

          {selectedEventId && !inviteLinks.length ? <div className="empty-state">No active invite links for this event.</div> : null}
          {inviteLinks.map((link) => (
            <article className="request-card" key={link.id}>
              <h4>{link.targetName || "Event Invite Link"}</h4>
              <p>Token: {link.code}</p>
              <p>Link: <a href={`/invite/event/${link.code}`} target="_blank" rel="noreferrer">/invite/event/{link.code}</a></p>
              <p>Uses: {link.usesCount || 0}{link.maxUses ? ` / ${link.maxUses}` : ""}</p>
              <div className="card-actions">
                <button className="btn-outline" onClick={() => deleteInviteLink(link.id)}>Deactivate</button>
              </div>
            </article>
          ))}
        </div>
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

function getObjectIdString(value) {
  if (!value) return "";
  if (typeof value === "string") return value;
  if (typeof value === "object" && typeof value.$oid === "string") return value.$oid;
  return "";
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

function ViewerPage() {
  const [tab, setTab] = useState("match");
  const [targetId, setTargetId] = useState("");
  const [payload, setPayload] = useState(null);
  const [error, setError] = useState("");

  async function loadData() {
    if (!targetId.trim()) return;
    setError("");
    let path = `/api/match/${targetId.trim()}`;
    if (tab === "rankings") {
      const [type, id] = targetId.split(":");
      path = fillRoute(ROUTE_TEMPLATES.publicRankings, { type: (type || "").trim(), id: (id || "").trim() });
    }
    if (tab === "tournament-fixtures") path = `/api/public/tournaments/${targetId.trim()}/fixtures`;
    if (tab === "tournament-standings") path = `/api/public/tournaments/${targetId.trim()}/standings`;
    if (tab === "championship") path = `/api/public/championships/${targetId.trim()}`;
    if (tab === "championship-fixtures") path = `/api/public/championships/${targetId.trim()}/fixtures`;
    if (tab === "championship-stats") path = `/api/public/championships/${targetId.trim()}/stats`;

    try {
      const response = await fetch(path);
      const data = await response.json().catch(() => ({}));
      if (!response.ok) throw new Error(data.error || `Failed to load ${tab}`);
      setPayload(data);
    } catch (err) {
      setError(err.message || "Failed to load data");
    }
  }

  return (
    <div className="dashboard-page">
      <header className="dashboard-navbar">
        <div className="brand">⚡ RaidX Viewer</div>
        <div className="navbar-actions">
          <button className="btn-outline" onClick={() => window.location.assign("/login")}>Back to Login</button>
        </div>
      </header>
      <main className="dashboard-container">
        <section className="panel-card">
          <div className="tab-row">
            {renderTab(tab, setTab, "match", "Match")}
            {renderTab(tab, setTab, "rankings", "Rankings")}
            {renderTab(tab, setTab, "tournament-fixtures", "Tournament Fixtures")}
            {renderTab(tab, setTab, "tournament-standings", "Tournament Standings")}
            {renderTab(tab, setTab, "championship", "Championship")}
            {renderTab(tab, setTab, "championship-fixtures", "Champ Fixtures")}
            {renderTab(tab, setTab, "championship-stats", "Champ Stats")}
          </div>
          <div className="form-grid">
            <label>Target ID {tab === "rankings" ? "(type:id)" : ""}</label>
            <input value={targetId} onChange={(event) => setTargetId(event.target.value)} placeholder={tab === "rankings" ? "tournament:EVENT_ID" : "event/match id"} />
            <button className="btn-primary-orange" onClick={loadData}>Load</button>
          </div>
          {error ? <div className="status-box danger">{error}</div> : null}
          {payload ? <pre style={{ whiteSpace: "pre-wrap" }}>{JSON.stringify(payload, null, 2)}</pre> : null}
        </section>
      </main>
    </div>
  );
}

function InviteLinkPage({ target }) {
  const { token } = useParams();
  const [details, setDetails] = useState(null);
  const [status, setStatus] = useState("");
  const [statusType, setStatusType] = useState("info");

  useEffect(() => {
    async function loadDetails() {
      try {
        const detailsTemplate = target === "team" ? ROUTE_TEMPLATES.teamInviteDetails : ROUTE_TEMPLATES.eventInviteDetails;
        const response = await fetch(fillRoute(detailsTemplate, { token }));
        const data = await response.json().catch(() => ({}));
        if (!response.ok) throw new Error(data.error || "Failed to load invite link details");
        setDetails(data);
      } catch (err) {
        setStatusType("danger");
        setStatus(err.message || "Failed to load invite link details");
      }
    }
    loadDetails();
  }, [target, token]);

  async function claimOrAccept(action) {
    try {
      const actionTemplate = target === "team"
        ? (action === "accept" ? ROUTE_TEMPLATES.teamInviteAccept : ROUTE_TEMPLATES.teamInviteClaim)
        : (action === "accept" ? ROUTE_TEMPLATES.eventInviteAccept : ROUTE_TEMPLATES.eventInviteClaim);
      const response = await authClient.apiFetch(fillRoute(actionTemplate, { token }), { method: "POST" });
      const data = await response.json().catch(() => ({}));
      if (!response.ok) throw new Error(data.error || `Failed to ${action} invite`);
      setStatusType("success");
      setStatus(data.message || `Invite ${action} successful`);
    } catch (err) {
      setStatusType("danger");
      setStatus(err.message || `Failed to ${action} invite`);
    }
  }

  return (
    <div className="dashboard-page">
      <header className="dashboard-navbar">
        <div className="brand">⚡ RaidX Invite</div>
        <div className="navbar-actions">
          <button className="btn-outline" onClick={() => window.location.assign("/login")}>Login</button>
        </div>
      </header>
      <main className="dashboard-container">
        <section className="panel-card">
          <h3>{target === "team" ? "Team" : "Event"} Invite Link</h3>
          {status ? <div className={`status-box ${statusType}`}>{status}</div> : null}
          {details ? <pre style={{ whiteSpace: "pre-wrap" }}>{JSON.stringify(details, null, 2)}</pre> : <p>Loading details...</p>}
          <div className="card-actions">
            <button className="btn-primary-orange" onClick={() => claimOrAccept("accept")}>Accept</button>
            <button className="btn-outline" onClick={() => claimOrAccept("claim")}>Claim</button>
          </div>
        </section>
      </main>
    </div>
  );
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
      <Route path="/signup" element={<SignupPage />} />
      <Route path="/" element={<Navigate to="/login" replace />} />
      <Route path="/viewer" element={<ViewerPage />} />
      <Route path="/invite/team/:token" element={<InviteLinkPage target="team" />} />
      <Route path="/invite/event/:token" element={<InviteLinkPage target="event" />} />
      <Route path="/player/dashboard" element={<Navigate to="/dashboard/player" replace />} />
      <Route path="/owner/dashboard" element={<Navigate to="/dashboard/team_owner" replace />} />
      <Route path="/organizer/dashboard" element={<Navigate to="/dashboard/organizer" replace />} />
      <Route path="/owner/teams" element={<Navigate to="/dashboard/team_owner" replace />} />
      <Route path="/owner/team/:id" element={<Navigate to="/dashboard/team_owner" replace />} />
      <Route path="/owner/team/:id/edit" element={<Navigate to="/dashboard/team_owner" replace />} />
      <Route path="/owner/teams/:id/pending-approvals" element={<Navigate to="/dashboard/team_owner" replace />} />
      <Route path="/owner/invite-links" element={<Navigate to="/dashboard/team_owner" replace />} />
      <Route path="/owner/match/:id/view" element={<Navigate to="/dashboard/team_owner" replace />} />
      <Route path="/owner/profile/:id" element={<Navigate to="/profile" replace />} />
      <Route path="/organizer/events" element={<Navigate to="/dashboard/organizer" replace />} />
      <Route path="/organizer/event/:id" element={<Navigate to="/dashboard/organizer" replace />} />
      <Route path="/organizer/tournament" element={<Navigate to="/dashboard/organizer" replace />} />
      <Route path="/organizer/championship" element={<Navigate to="/dashboard/organizer" replace />} />
      <Route path="/organizer/event/:id/matches" element={<Navigate to="/dashboard/organizer" replace />} />
      <Route path="/organizer/match/:id/teams" element={<Navigate to="/dashboard/organizer" replace />} />
      <Route path="/organizer/match/:id/stats" element={<Navigate to="/dashboard/organizer" replace />} />
      <Route path="/organizer/match/:id/scorer" element={<Navigate to="/dashboard/organizer" replace />} />
      <Route path="/organizer/playerselection/:id" element={<Navigate to="/dashboard/organizer" replace />} />
      <Route path="/organizer/events/:id/pending-approvals" element={<Navigate to="/dashboard/organizer" replace />} />
      <Route path="/organizer/event/:id/pending-approvals" element={<Navigate to="/dashboard/organizer" replace />} />
      <Route path="/organizer/invite-links" element={<Navigate to="/dashboard/organizer" replace />} />
      <Route path="/organizer/match/:id/view" element={<Navigate to="/dashboard/organizer" replace />} />
      <Route path="/organizer/profile/:id" element={<Navigate to="/profile" replace />} />
      <Route path="/playerprofile/:id" element={<Navigate to="/profile" replace />} />
      <Route path="/rankings/:type/:id" element={<Navigate to="/viewer" replace />} />
      <Route path="/viewer/match/:id" element={<Navigate to="/viewer" replace />} />
      <Route path="/viewer/match/:id/overview" element={<Navigate to="/viewer" replace />} />
      <Route path="/viewer/tournament/:id" element={<Navigate to="/viewer" replace />} />
      <Route path="/viewer/championship/:id" element={<Navigate to="/viewer" replace />} />
      <Route path="/scorer" element={<Navigate to="/dashboard/organizer" replace />} />
      <Route path="/endgame" element={<Navigate to="/dashboard/organizer" replace />} />
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
