export function normalizeRole(role) {
  const normalized = String(role || "player").toLowerCase();
  if (normalized === "owner") return "team_owner";
  return normalized;
}

export function normalizeStatus(invite) {
  return String(invite?.status || invite?.Status || "pending").toLowerCase();
}

export function getInviteId(invite) {
  return invite?.id || invite?.ID || invite?._id || "";
}

export function getDeclineReason(invite) {
  return invite?.declineReason || invite?.decline_reason || invite?.DeclineReason || "";
}

export function getStatusDisplay(statusValue) {
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

export function safeArray(data) {
  return Array.isArray(data) ? data : [];
}

export function getTeamId(team) {
  return team?.ID || team?.id || team?.teamId || team?.TeamID || "";
}

export function getObjectIdString(value) {
  if (!value) return "";
  if (typeof value === "string") return value;
  if (typeof value === "object" && typeof value.$oid === "string") return value.$oid;
  return "";
}
