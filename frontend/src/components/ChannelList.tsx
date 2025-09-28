type Channel = { id: string; name: string; is_private?: boolean };
type Props = {
  channels: Channel[];
  activeId: string | null;
  onPick: (id: string) => void;
};

export default function ChannelList({ channels, activeId, onPick }: Props) {
  return (
    <div className="list">
      {channels.length === 0 && (
        <div style={{ color: "#9ca3af" }}>No channels</div>
      )}
      {channels.map((ch) => {
        const isPrivate = !!ch.is_private;
        const icon = isPrivate ? "🔒" : "🌐";
        const chipLabel = isPrivate ? "Private" : "Public";

        return (
          <div
            key={ch.id}
            className={`list-item ${activeId === ch.id ? "active" : ""}`}
            onClick={() => onPick(ch.id)}
            title={`${ch.name} (${chipLabel})`}
            role="button"
            tabIndex={0}
            onKeyDown={(e) => {
              if (e.key === "Enter" || e.key === " ") onPick(ch.id);
            }}
            style={{ display: "flex", alignItems: "center", gap: 8 }}
          >
            <span aria-hidden>{icon}</span>
            <span style={{ flex: 1, overflow: "hidden", textOverflow: "ellipsis" }}>
              # {ch.name}
            </span>
            <span
              className="chip"
              style={{
                fontSize: 12,
                padding: "2px 6px",
                borderRadius: 999,
                background: isPrivate ? "#fee2e2" : "#e0f2fe",
                color: isPrivate ? "#991b1b" : "#075985",
                whiteSpace: "nowrap",
              }}
            >
              {chipLabel}
            </span>
          </div>
        );
      })}
    </div>
  );
}
