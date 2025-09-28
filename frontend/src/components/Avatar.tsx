type AvatarProps = {
  src?: string | null;
  name?: string | null;
  size?: number;                 // px
  shape?: "circle" | "rounded";  // 形
  mode?: "contain" | "cover";    // contain = 縮小のみ（非トリミング）
  bg?: string;
};

export default function Avatar({
  src,
  name,
  size = 32,
  shape = "circle",
  mode = "contain",
  bg = "#f3f4f6",
}: AvatarProps) {
  const borderRadius = shape === "circle" ? "50%" : "10px";
  const initials =
    (name || "")
      .trim()
      .split(/\s+/)
      .map((s) => s[0]?.toUpperCase())
      .join("")
      .slice(0, 2) || "U";

  if (src) {
    return (
      <img
        src={src}
        alt={name || ""}
        width={size}
        height={size}
        style={{
          width: size,
          height: size,
          borderRadius,
          objectFit: mode,
          background: bg,
          display: "block",
        }}
        loading="lazy"
      />
    );
  }

  return (
    <div
      style={{
        width: size,
        height: size,
        borderRadius,
        background: bg,
        color: "#374151",
        display: "grid",
        placeItems: "center",
        fontSize: size * 0.45,
        fontWeight: 700,
        userSelect: "none",
      }}
      aria-label={name || "User"}
      title={name || "User"}
    >
      {initials}
    </div>
  );
}
