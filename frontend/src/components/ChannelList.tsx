export type Channel = {
  id: string;
  name: string;
};

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
      {channels.map((ch) => (
        <div
          key={ch.id}
          className={`list-item ${activeId === ch.id ? "active" : ""}`}
          onClick={() => onPick(ch.id)}
          title={ch.id}
          role="button"
          tabIndex={0}
          onKeyDown={(e) => {
            if (e.key === "Enter" || e.key === " ") onPick(ch.id);
          }}
        >
          # {ch.name}
        </div>
      ))}
    </div>
  );
}
