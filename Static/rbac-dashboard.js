document.addEventListener('DOMContentLoaded', () => {
    const role = (getRoleFromToken() || '').toLowerCase();
    const pageRole = (document.body.dataset.role || '').toLowerCase();

    const roleBadge = document.getElementById('role-badge');
    if (roleBadge) {
        roleBadge.textContent = role ? role.replace('_', ' ') : 'unknown';
    }

    if (pageRole && role && pageRole !== role) {
        const warn = document.getElementById('role-warning');
        if (warn) {
            warn.textContent = `You are logged in as ${role}. This page is for ${pageRole}.`;
            warn.classList.remove('d-none');
        }
    }

    if (pageRole === 'player') {
        initPlayerDashboard();
    }
    if (pageRole === 'team_owner') {
        initOwnerDashboard();
    }
    if (pageRole === 'organizer') {
        initOrganizerDashboard();
    }
});

function setStatus(id, text, type = 'info') {
    const el = document.getElementById(id);
    if (!el) return;
    el.className = `alert alert-${type}`;
    el.textContent = text;
    el.classList.remove('d-none');
}

function hideStatus(id) {
    const el = document.getElementById(id);
    if (!el) return;
    el.classList.add('d-none');
}

function formatId(id) {
    if (!id) return '';
    return typeof id === 'string' ? id : (id.$oid || id.Hex || id.hex || JSON.stringify(id));
}

function getInviteId(invite) {
    return invite?.ID || invite?._id || invite?.id || invite?.Id || '';
}

function getInviteField(invite, camelName, snakeName) {
    return invite?.[camelName] || invite?.[snakeName] || '';
}

async function initPlayerDashboard() {
    const profileLink = document.getElementById('player-profile-link');
    if (profileLink) {
        const userId = getUserIdFromToken();
        profileLink.href = userId ? `/playerprofile/${userId}` : '#';
    }
    const refreshBtn = document.getElementById('player-refresh');
    if (refreshBtn) {
        refreshBtn.addEventListener('click', loadPlayerInvites);
    }
    await loadPlayerInvites();
}

async function loadPlayerInvites() {
    hideStatus('player-status');
    const pendingList = document.getElementById('player-invites-pending');
    const acceptedList = document.getElementById('player-invites-accepted');
    const declinedList = document.getElementById('player-invites-declined');
    if (!pendingList || !acceptedList || !declinedList) return;
    pendingList.innerHTML = '';
    acceptedList.innerHTML = '';
    declinedList.innerHTML = '';

    try {
        const res = await apiRequest('/api/invitations');
        const data = await res.json();
        if (!Array.isArray(data) || data.length === 0) {
            pendingList.innerHTML = '<div class="text-white">No pending invitations.</div>';
            acceptedList.innerHTML = '<div class="text-white">No accepted invitations.</div>';
            declinedList.innerHTML = '<div class="text-white">No declined invitations.</div>';
            return;
        }
        const pendingInvites = data.filter(inv => (inv.status || inv.Status) === 'pending');
        const acceptedInvites = data.filter(inv => (inv.status || inv.Status) === 'accepted');
        const declinedInvites = data.filter(inv => (inv.status || inv.Status) === 'declined');

        renderPlayerInviteList(pendingList, pendingInvites, true);
        renderPlayerInviteList(acceptedList, acceptedInvites, false);
        renderPlayerInviteList(declinedList, declinedInvites, false);
    } catch (e) {
        setStatus('player-status', 'Failed to load invitations', 'danger');
    }
}

function renderPlayerInviteList(container, invites, showActions) {
    if (!container) return;
    if (!invites || invites.length === 0) {
        container.innerHTML = '<div class="text-white">No invitations.</div>';
        return;
    }

    invites.forEach(invite => {
            const card = document.createElement('div');
            card.className = 'invite-card mb-3';
            const inviteId = invite?.id || getInviteId(invite);
        const statusValue = invite.status || invite.Status || 'pending';
            card.innerHTML = `
                <div class="d-flex justify-content-between align-items-start flex-wrap gap-3">
                    <div>
                        <div class="fw-semibold fs-5">Team Invite</div>
                        <div class="invite-meta">Team: ${invite.teamName || 'Unknown'}</div>
                        <div class="invite-meta">Owner: ${invite.ownerName || 'Unknown'}</div>
                        <div class="invite-meta">Invite ID: ${formatId(inviteId)}</div>
                        <div class="invite-meta">Team ID: ${formatId(invite.teamId || getInviteField(invite, 'TeamID', 'team_id'))}</div>
                    <div class="invite-meta">Status: ${statusValue}</div>
                    </div>
                ${showActions ? `
                <div class="d-flex gap-2">
                    <button class="btn btn-sm btn-success">Accept</button>
                    <button class="btn btn-sm btn-outline-danger">Decline</button>
                </div>
                ` : ''}
                </div>
            `;
        if (showActions) {
            const [acceptBtn, declineBtn] = card.querySelectorAll('button');
            acceptBtn.addEventListener('click', () => updateInvite(inviteId, 'accepted'));
            declineBtn.addEventListener('click', () => updateInvite(inviteId, 'declined'));
        }
        container.appendChild(card);
    });
}

async function updateInvite(inviteId, status, extra = {}) {
    try {
        if (!inviteId) {
            throw new Error('Invalid invitation id');
        }
        const url = `/api/invitations/${formatId(inviteId)}`;
        const res = await apiRequest(url, {
            method: 'PUT',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ status, ...extra })
        });
        if (!res.ok) {
            const err = await res.json();
            throw new Error(err.error || `Failed to update invite (${res.status})`);
        }
        const responseData = await res.json();
        await loadPlayerInvites();
        if (document.getElementById('owner-event-invites')) {
            await loadOwnerEventInvites();
        }
        setStatus('player-status', 'Invitation updated successfully', 'success');
    } catch (e) {
        console.error('DEBUG updateInvite - error:', e);
        setStatus('player-status', e.message || 'Failed to update invitation', 'danger');
    }
}

async function initOwnerDashboard() {
    const teamIdInput = document.getElementById('owner-team-id');
    const storedTeamId = localStorage.getItem('rbac_team_id');
    if (teamIdInput && storedTeamId) teamIdInput.value = storedTeamId;

    const createForm = document.getElementById('owner-create-team');
    if (createForm) {
        createForm.addEventListener('submit', async (e) => {
            e.preventDefault();
            hideStatus('owner-status');
            const teamName = document.getElementById('team-name').value.trim();
            const description = document.getElementById('team-desc').value.trim();
            try {
                const res = await apiRequest('/api/teams', {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify({ team_name: teamName, description })
                });
                const data = await res.json();
                if (!res.ok) throw new Error(data.error || 'Failed to create team');
                const newTeamId = formatId(data.team_id);
                if (teamIdInput) teamIdInput.value = newTeamId;
                localStorage.setItem('rbac_team_id', newTeamId);
                setStatus('owner-status', `Team created: ${data.team_name} (${newTeamId})`, 'success');
            } catch (e) {
                setStatus('owner-status', e.message, 'danger');
            }
        });
    }

    const inviteForm = document.getElementById('owner-invite-player');
    if (inviteForm) {
        inviteForm.addEventListener('submit', async (e) => {
            e.preventDefault();
            hideStatus('owner-status');
            const teamId = (teamIdInput && teamIdInput.value.trim()) || '';
            const playerId = document.getElementById('player-id').value.trim();
            const username = document.getElementById('player-username').value.trim();
            const generateLink = document.getElementById('generate-link').checked;
            try {
                const res = await apiRequest(`/api/teams/${teamId}/invite`, {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify({ player_id: playerId, username, generate_link: generateLink })
                });
                const data = await res.json();
                if (!res.ok) throw new Error(data.error || 'Failed to invite player');
                const linkBox = document.getElementById('invite-link');
                if (linkBox) linkBox.value = data.invite_url || '';
                setStatus('owner-status', 'Invitation created.', 'success');
            } catch (e) {
                setStatus('owner-status', e.message, 'danger');
            }
        });
    }

    const refreshInvitesBtn = document.getElementById('owner-refresh');
    if (refreshInvitesBtn) refreshInvitesBtn.addEventListener('click', loadOwnerTeamInvites);
    await loadOwnerTeamInvites();

    const refreshEventInvitesBtn = document.getElementById('owner-event-refresh');
    if (refreshEventInvitesBtn) refreshEventInvitesBtn.addEventListener('click', loadOwnerEventInvites);
    await loadOwnerEventInvites();
}

async function loadOwnerTeamInvites() {
    const list = document.getElementById('owner-invites');
    const teamId = document.getElementById('owner-team-id')?.value.trim();
    if (!list || !teamId) return;
    list.innerHTML = '';

    try {
        const res = await apiRequest(`/api/teams/${teamId}/invites`);
        const data = await res.json();
        if (!Array.isArray(data) || data.length === 0) {
            list.innerHTML = '<div class="text-white">No invites found for this team.</div>';
            return;
        }
        data.forEach(invite => {
            const card = document.createElement('div');
            card.className = 'card mb-2';
            const inviteId = getInviteId(invite);
            card.innerHTML = `
                <div class="card-body">
                    <div class="fw-semibold">Player Invite</div>
                    <div class="small text-white">Invite ID: ${formatId(inviteId)}</div>
                    <div class="small text-white">Player ID: ${formatId(getInviteField(invite, 'ToID', 'to_id'))}</div>
                    <div class="small text-white">Status: ${getInviteField(invite, 'Status', 'status')}</div>
                </div>
            `;
            list.appendChild(card);
        });
    } catch (e) {
        setStatus('owner-status', 'Failed to load team invites', 'danger');
    }
}

async function loadOwnerEventInvites() {
    const list = document.getElementById('owner-event-invites');
    if (!list) return;
    list.innerHTML = '';

    let ownerTeams = [];
    try {
        const teamsRes = await apiRequest('/api/owner/teams');
        ownerTeams = await teamsRes.json();
    } catch (e) {
        ownerTeams = [];
    }

    try {
        // Load ALL event invitations (not just pending)
        const res = await apiRequest('/api/owner/event-invitations');
        const data = await res.json();
        console.log('Event invitations response:', data); // Debug log
        console.log('Response status:', res.status, 'Is array?', Array.isArray(data)); // More debug
        if (!res.ok) {
            throw new Error(data.error || 'Failed to load event invites');
        }
        if (!Array.isArray(data) || data.length === 0) {
            list.innerHTML = '<div class="text-white">No event invites.</div>';
            return;
        }
        
        // Separate by status
        const pending = data.filter(inv => inv.status === 'pending');
        const accepted = data.filter(inv => inv.status === 'accepted');
        const declined = data.filter(inv => inv.status === 'declined');
        
        // Render pending first
        if (pending.length > 0) {
            const pendingHeader = document.createElement('h6');
            pendingHeader.className = 'text-warning mt-3 mb-2';
            pendingHeader.textContent = 'Pending Invitations';
            list.appendChild(pendingHeader);
        }
        pending.forEach(invite => {
            renderOwnerEventInvite(list, invite, ownerTeams, true);
        });
        
        // Render accepted
        if (accepted.length > 0) {
            const acceptedHeader = document.createElement('h6');
            acceptedHeader.className = 'text-success mt-3 mb-2';
            acceptedHeader.textContent = 'Accepted';
            list.appendChild(acceptedHeader);
        }
        accepted.forEach(invite => {
            renderOwnerEventInvite(list, invite, ownerTeams, false);
        });
        
        // Render declined
        if (declined.length > 0) {
            const declinedHeader = document.createElement('h6');
            declinedHeader.className = 'text-danger mt-3 mb-2';
            declinedHeader.textContent = 'Declined';
            list.appendChild(declinedHeader);
        }
        declined.forEach(invite => {
            renderOwnerEventInvite(list, invite, ownerTeams, false);
        });
        
        if (pending.length === 0 && accepted.length === 0 && declined.length === 0) {
            list.innerHTML = '<div class="text-white">No event invites.</div>';
        }
    } catch (e) {
        setStatus('owner-status', 'Failed to load event invites', 'danger');
    }
}

function renderOwnerEventInvite(list, invite, ownerTeams, showActions) {
    const inviteId = getInviteId(invite);
    console.log('DEBUG renderOwnerEventInvite - Processing invite, ID:', inviteId, 'Full invite:', invite);
    const card = document.createElement('div');
    card.className = 'card mb-2';
    card.setAttribute('data-invite-id', inviteId); // Add for debugging
    const statusBadge = invite.status === 'pending' ? 'badge bg-warning' : 
                       invite.status === 'accepted' ? 'badge bg-success' : 'badge bg-danger';
    const eventName = invite.eventName || invite.EventName || 'Unknown Event';
    const eventType = invite.eventType || invite.EventType || 'Unknown Type';
    const ownerName = invite.ownerName || invite.OwnerName || 'Unknown Owner';
    const teamName = invite.teamName || invite.TeamName || 'Unassigned';
    card.innerHTML = `
                <div class="card-body">
                    <div class="fw-semibold">Event Invite <span class="${statusBadge}">${invite.status}</span></div>
                    <div class="small text-white">Team Owner: ${ownerName}</div>
                    <div class="small text-white">Event: ${eventName}</div>
                    <div class="small text-white">Event Type: ${eventType}</div>
                    <div class="small text-white">Team: ${teamName}</div>
                    ${showActions ? `<div class="mt-2">
                        <select class="form-select form-select-sm mb-2" style="max-width: 220px;">
                            <option value="">Select team to accept...</option>
                            ${ownerTeams.map(t => `<option value="${t.ID}">${t.TeamName}</option>`).join('')}
                        </select>
                        <button class="btn btn-sm btn-success me-2" data-action="accept">Accept</button>
                        <button class="btn btn-sm btn-outline-danger" data-action="decline">Decline</button>
                    </div>` : ''}
                </div>
            `;
    if (showActions) {
        const selectEl = card.querySelector('select');
        const acceptBtn = card.querySelector('button[data-action="accept"]');
        const declineBtn = card.querySelector('button[data-action="decline"]');
        acceptBtn.addEventListener('click', () => {
            console.log('DEBUG Accept clicked for inviteId:', inviteId);
            const selectedTeamId = selectEl ? selectEl.value.trim() : '';
            if (!selectedTeamId) {
                setStatus('owner-status', 'Select a team before accepting.', 'warning');
                return;
            }
            console.log('DEBUG Calling updateInvite with inviteId:', inviteId, 'team_id:', selectedTeamId);
            updateInvite(inviteId, 'accepted', { team_id: selectedTeamId });
        });
        declineBtn.addEventListener('click', () => {
            console.log('DEBUG Decline clicked for inviteId:', inviteId);
            updateInvite(inviteId, 'declined');
        });
    }
    list.appendChild(card);
}

async function initOrganizerDashboard() {
    const profileLink = document.getElementById('organizer-profile-link');
    if (profileLink) {
        const userId = getUserIdFromToken();
        profileLink.href = userId ? `/organizer/profile/${userId}` : '#';
    }

    const eventTypeSelect = document.getElementById('event-type');
    const maxTeamsGroup = document.getElementById('max-teams-group');
    const maxTeamsInput = document.getElementById('event-max-teams');
    if (eventTypeSelect && maxTeamsGroup) {
        const toggleMaxTeams = () => {
            const type = eventTypeSelect.value;
            const shouldShow = (type === 'tournament' || type === 'championship');
            maxTeamsGroup.style.display = shouldShow ? 'block' : 'none';
            if (maxTeamsInput) {
                maxTeamsInput.required = shouldShow;
            }
        };
        eventTypeSelect.addEventListener('change', toggleMaxTeams);
        toggleMaxTeams();
    }

    const eventIdInput = document.getElementById('organizer-event-id');
    const storedEventId = localStorage.getItem('rbac_event_id');
    if (eventIdInput && storedEventId) eventIdInput.value = storedEventId;

    const createForm = document.getElementById('organizer-create-event');
    if (createForm) {
        createForm.addEventListener('submit', async (e) => {
            e.preventDefault();
            hideStatus('organizer-status');
            const name = document.getElementById('event-name').value.trim();
            const type = document.getElementById('event-type').value;
            const maxTeams = parseInt(document.getElementById('event-max-teams')?.value || '0', 10);
            if ((type === 'tournament' || type === 'championship') && maxTeams <= 0) {
                setStatus('organizer-status', 'Please enter number of teams for tournaments/championships.', 'warning');
                return;
            }
            try {
                const res = await apiRequest('/api/events', {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify({ event_name: name, event_type: type, max_teams: maxTeams })
                });
                const data = await res.json();
                if (!res.ok) throw new Error(data.error || 'Failed to create event');
                const newEventId = formatId(data.event_id);
                if (eventIdInput) eventIdInput.value = newEventId;
                localStorage.setItem('rbac_event_id', newEventId);
                setStatus('organizer-status', `Event created: ${data.event_name} (${newEventId})`, 'success');
                await loadOrganizerEvents();
            } catch (e) {
                setStatus('organizer-status', e.message, 'danger');
            }
        });
    }
    const refreshEventsBtn = document.getElementById('organizer-refresh-events');
    if (refreshEventsBtn) refreshEventsBtn.addEventListener('click', loadOrganizerEvents);
    await loadOrganizerEvents();

    const refreshRequestsBtn = document.getElementById('organizer-refresh-requests');
    if (refreshRequestsBtn) refreshRequestsBtn.addEventListener('click', loadOrganizerRequests);
    await loadOrganizerRequests();
}

async function loadOrganizerEvents() {
    const list = document.getElementById('organizer-events');
    const completedList = document.getElementById('organizer-events-completed');
    if (!list || !completedList) {
        console.error('DEBUG: Missing organizer event containers');
        return;
    }
    list.innerHTML = '';
    completedList.innerHTML = '';

    try {
        const res = await apiRequest('/api/organizer/events');
        const data = await res.json();
        console.log('DEBUG loadOrganizerEvents - Response status:', res.status);
        console.log('DEBUG loadOrganizerEvents - Data:', data);
        if (!res.ok) {
            throw new Error(data.error || 'Failed to load events');
        }
        if (!Array.isArray(data) || data.length === 0) {
            console.log('DEBUG: No events found');
            list.innerHTML = '<div class="text-white">No events created yet.</div>';
            completedList.innerHTML = '<div class="text-white">No completed events.</div>';
            return;
        }

        console.log('DEBUG: Found', data.length, 'total events');
        const activeEvents = data.filter(evt => evt.status !== 'completed');
        const completedEvents = data.filter(evt => evt.status === 'completed');
        console.log('DEBUG: Active:', activeEvents.length, 'Completed:', completedEvents.length);

        renderOrganizerEventList(list, activeEvents, true);
        renderOrganizerEventList(completedList, completedEvents, false);
    } catch (e) {
        console.error('DEBUG loadOrganizerEvents error:', e);
        setStatus('organizer-status', e.message || 'Failed to load events', 'danger');
    }
}

function renderOrganizerEventList(container, events, allowActions) {
    if (!events || events.length === 0) {
        container.innerHTML = '<div class="text-white">No events available.</div>';
        return;
    }

    events.forEach(event => {
        const card = document.createElement('div');
        card.className = 'event-card';
        card.innerHTML = `
            <h5>${event.eventName}</h5>
            <p><strong>Type:</strong> ${event.eventType}</p>
            ${event.maxTeams ? `<p><strong>Max Teams:</strong> ${event.maxTeams}</p>` : ''}
            <p><strong>Status:</strong> <span class="badge-custom">${event.status === 'completed' ? 'done' : event.status}</span></p>
            <p><strong>Accepted:</strong> ${event.counts?.accepted || 0} | <strong>Pending:</strong> ${event.counts?.pending || 0} | <strong>Declined:</strong> ${event.counts?.declined || 0}</p>
            ${allowActions ? `
            <div class="event-card-actions">
                <button class="btn btn-primary-orange btn-sm" onclick="populateEventEdit('${event.id}')">Edit</button>
                <button class="btn btn-outline-light btn-sm" onclick="toggleInviteForm('${event.id}')">Invite Team Owner</button>
                <button class="btn btn-success btn-sm" onclick="markEventDone('${event.id}')">Mark as Done</button>
            </div>
            <div class="mt-3" id="event-edit-${event.id}" style="display:none;">
                <div class="row g-2">
                    <div class="col-md-5">
                        <input class="form-control" value="${event.eventName}" id="edit-name-${event.id}" />
                    </div>
                    <div class="col-md-3">
                        <select class="form-select" id="edit-type-${event.id}">
                            <option value="match" ${event.eventType === 'match' ? 'selected' : ''}>Match</option>
                            <option value="tournament" ${event.eventType === 'tournament' ? 'selected' : ''}>Tournament</option>
                            <option value="championship" ${event.eventType === 'championship' ? 'selected' : ''}>Championship</option>
                        </select>
                    </div>
                    <div class="col-md-2">
                        <input class="form-control" type="number" min="0" id="edit-max-${event.id}" value="${event.maxTeams || 0}" />
                    </div>
                    <div class="col-md-2 d-grid">
                        <button class="btn btn-sm btn-success" onclick="saveEventEdit('${event.id}')">Save</button>
                    </div>
                </div>
            </div>
            <div class="mt-3" id="event-invite-${event.id}" style="display:none;">
                <div class="row g-2">
                    <div class="col-md-6">
                        <input class="form-control" placeholder="Team owner username or email" id="invite-owner-${event.id}" />
                    </div>
                    <div class="col-md-3 d-grid">
                        <button class="btn btn-primary-orange btn-sm" onclick="sendOrganizerInvite('${event.id}')">Send Invite</button>
                    </div>
                </div>
            </div>
            ` : ''}
        `;
        container.appendChild(card);
    });
}

function populateEventEdit(eventId) {
    const editSection = document.getElementById(`event-edit-${eventId}`);
    if (editSection) editSection.style.display = editSection.style.display === 'none' ? 'block' : 'none';
}

function toggleInviteForm(eventId) {
    const form = document.getElementById(`event-invite-${eventId}`);
    if (form) form.style.display = form.style.display === 'none' ? 'block' : 'none';
}

async function saveEventEdit(eventId) {
    const name = document.getElementById(`edit-name-${eventId}`)?.value.trim();
    const type = document.getElementById(`edit-type-${eventId}`)?.value;
    const maxTeams = parseInt(document.getElementById(`edit-max-${eventId}`)?.value || '0', 10);
    if (!name) return;
    if ((type === 'tournament' || type === 'championship') && maxTeams <= 0) {
        setStatus('organizer-status', 'Please enter number of teams for tournaments/championships.', 'warning');
        return;
    }

    try {
        const res = await apiRequest(`/api/events/${eventId}`, {
            method: 'PUT',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ event_name: name, event_type: type, max_teams: maxTeams })
        });
        if (!res.ok) {
            const err = await res.json();
            throw new Error(err.error || 'Failed to update event');
        }
        await loadOrganizerEvents();
    } catch (e) {
        setStatus('organizer-status', e.message, 'danger');
    }
}

async function sendOrganizerInvite(eventId) {
    const ownerInput = document.getElementById(`invite-owner-${eventId}`);
    const ownerIdentifier = ownerInput ? ownerInput.value.trim() : '';
    if (!ownerIdentifier) {
        setStatus('organizer-status', 'Enter team owner username or email.', 'warning');
        return;
    }

    try {
        const res = await apiRequest(`/api/events/${eventId}/invite`, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ ownerIdentifier })
        });
        const data = await res.json();
        if (!res.ok) throw new Error(data.error || 'Failed to invite team owner');
        setStatus('organizer-status', 'Invitation created.', 'success');
        if (ownerInput) ownerInput.value = '';
        await loadOrganizerRequests();
    } catch (e) {
        setStatus('organizer-status', e.message, 'danger');
    }
}

async function markEventDone(eventId) {
    try {
        const res = await apiRequest(`/api/events/${eventId}/complete`, {
            method: 'POST'
        });
        const data = await res.json();
        if (!res.ok) throw new Error(data.error || 'Failed to update event');
        await loadOrganizerEvents();
    } catch (e) {
        setStatus('organizer-status', e.message, 'danger');
    }
}

async function loadOrganizerRequests() {
    const pendingList = document.getElementById('organizer-requests-pending');
    const acceptedList = document.getElementById('organizer-requests-accepted');
    const declinedList = document.getElementById('organizer-requests-declined');
    if (!pendingList || !acceptedList || !declinedList) {
        console.error('DEBUG: Missing organizer request containers');
        return;
    }
    pendingList.innerHTML = '';
    acceptedList.innerHTML = '';
    declinedList.innerHTML = '';

    try {
        const res = await apiRequest('/api/organizer/event-invites');
        const data = await res.json();
        console.log('DEBUG loadOrganizerRequests - Response status:', res.status);
        console.log('DEBUG loadOrganizerRequests - Data:', data);
        if (!res.ok) {
            throw new Error(data.error || 'Failed to load requests');
        }
        if (!Array.isArray(data) || data.length === 0) {
            console.log('DEBUG: No requests found');
            pendingList.innerHTML = '<div class="text-white">No pending requests.</div>';
            acceptedList.innerHTML = '<div class="text-white">No accepted requests.</div>';
            declinedList.innerHTML = '<div class="text-white">No declined requests.</div>';
            return;
        }

        console.log('DEBUG: Found', data.length, 'total requests');
        const pending = data.filter(inv => inv.status === 'pending');
        const accepted = data.filter(inv => inv.status === 'accepted');
        const declined = data.filter(inv => inv.status === 'declined');
        console.log('DEBUG: Pending:', pending.length, 'Accepted:', accepted.length, 'Declined:', declined.length);

        renderOrganizerRequestList(pendingList, pending);
        renderOrganizerRequestList(acceptedList, accepted);
        renderOrganizerRequestList(declinedList, declined);
    } catch (e) {
        console.error('DEBUG loadOrganizerRequests error:', e);
        setStatus('organizer-status', e.message || 'Failed to load requests', 'danger');
    }
}

function renderOrganizerRequestList(container, requests) {
    if (!requests || requests.length === 0) {
        container.innerHTML = '<div class="text-muted">No requests.</div>';
        return;
    }
    requests.forEach(req => {
        const card = document.createElement('div');
        card.className = 'request-card';
        const teamName = req.teamName || 'Unassigned';
        const teamId = req.teamId || 'Unassigned';
        card.innerHTML = `
            <h6>${req.eventName || 'Event'}</h6>
            <p class="mb-1">Team Owner: ${req.ownerName || 'Unknown'}</p>
            <p class="mb-1">User ID: ${req.ownerUserId || 'N/A'}</p>
            <p class="mb-1">Team: <span class="text-white">${teamName}</span></p>
            <p class="mb-1">Team ID: <span class="text-white">${teamId}</span></p>
            <p>Status: <span class="badge-custom">${req.status}</span></p>
            <p class="text-muted mb-0">Invite ID: ${req.id}</p>
        `;
        container.appendChild(card);
    });
}

async function loadOrganizerEventTeams() {
    const list = document.getElementById('organizer-event-teams');
    const eventId = document.getElementById('organizer-event-id')?.value.trim();
    if (!list || !eventId) return;
    list.innerHTML = '';

    try {
        const res = await apiRequest(`/api/events/${eventId}/teams`);
        const data = await res.json();
        if (!Array.isArray(data) || data.length === 0) {
            list.innerHTML = '<div class="text-muted">No teams linked to this event.</div>';
            return;
        }
        data.forEach(entry => {
            const card = document.createElement('div');
            card.className = 'card mb-2';
            card.innerHTML = `
                <div class="card-body">
                    <div class="fw-semibold">Team: ${formatId(entry.team_id)}</div>
                    <div class="small text-muted">Status: ${entry.status}</div>
                </div>
            `;
            list.appendChild(card);
        });
    } catch (e) {
        setStatus('organizer-status', 'Failed to load event teams', 'danger');
    }
}
