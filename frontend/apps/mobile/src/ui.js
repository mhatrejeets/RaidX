import { Text, TouchableOpacity, View } from "react-native";

export function Button({ label, onPress, variant = "light", disabled = false }) {
  const backgroundColor = variant === "primary" ? "#f97316" : variant === "danger" ? "#ef4444" : "#334155";
  return (
    <TouchableOpacity disabled={disabled} onPress={onPress} style={{ padding: 12, marginBottom: 8, backgroundColor, borderRadius: 8, opacity: disabled ? 0.65 : 1 }}>
      <Text style={{ color: "#fff", fontWeight: "600" }}>{label}</Text>
    </TouchableOpacity>
  );
}

export function TabRow({ items, current, onChange }) {
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

export function SectionCard({ title, subtitle, children }) {
  return (
    <View style={{ backgroundColor: "#1e293b", borderRadius: 12, borderWidth: 1, borderColor: "#334155", padding: 12, marginBottom: 12 }}>
      <Text style={{ color: "#fbbf24", fontSize: 18, fontWeight: "700" }}>{title}</Text>
      {subtitle ? <Text style={{ color: "#cbd5e1", marginBottom: 10 }}>{subtitle}</Text> : null}
      {children}
    </View>
  );
}

export function StatusBox({ message, type = "info" }) {
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

export function ListItem({ title, children }) {
  return (
    <View style={{ backgroundColor: "#0f172a", borderColor: "#334155", borderWidth: 1, borderRadius: 10, padding: 10, marginBottom: 8 }}>
      <Text style={{ color: "#fbbf24", fontWeight: "700", marginBottom: 4 }}>{title}</Text>
      {children}
    </View>
  );
}
