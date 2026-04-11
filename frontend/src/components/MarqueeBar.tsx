import "@/styles/marquee.css";

/**
 * Props for the MarqueeBar component.
 * `position` controls whether the bar is anchored to the top or bottom
 * of the player container.
 */
interface MarqueeBarProps {
  /** Static label displayed on the left side of the bar (e.g. "Now Playing"). */
  label: string;
  /** Scrolling marquee text displayed to the right of the label. */
  text: string;
  /** Whether the bar is positioned at the top or bottom of the player. */
  position: "top" | "bottom";
}

/**
 * Renders a semi-transparent bar with a static label on the left and a
 * CSS-animated scrolling marquee text on the right. Used as an overlay
 * on the viewer video player for "Now Playing" / "Up Next" information.
 */
export function MarqueeBar({ label, text, position }: MarqueeBarProps) {
  return (
    <div
      className={`absolute left-0 right-0 z-10 flex items-center bg-black/80 px-4 py-2 ${
        position === "top" ? "top-0" : "bottom-0"
      }`}
    >
      {label && (
        <span className="mr-4 shrink-0 text-sm font-semibold text-blue-400">
          {label}
        </span>
      )}
      <div className="min-w-0 flex-1 overflow-hidden">
        <span className="marquee-text inline-block text-sm text-white">
          {text}
        </span>
      </div>
    </div>
  );
}
