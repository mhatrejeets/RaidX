export function renderTab(current, setCurrent, value, label) {
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
