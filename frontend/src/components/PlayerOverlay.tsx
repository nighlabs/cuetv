import { MarqueeBar } from "./MarqueeBar";
import type { RoomConfig, QueueItem } from "@/types/api";

/** Props for the PlayerOverlay component. */
interface PlayerOverlayProps {
  /** The current room configuration (marquee enabled/disabled, labels, sources). */
  config: RoomConfig;
  /** The full queue array, used to resolve marquee text by index. */
  queue: QueueItem[];
}

/**
 * Conditionally renders top and/or bottom marquee bars over the video player
 * based on the room configuration toggles (topMarqueeEnabled, bottomMarqueeEnabled).
 */
export function PlayerOverlay({ config, queue }: PlayerOverlayProps) {
  /**
   * Resolves the marquee display text from the queue based on the configured
   * source. "current" uses the video at currentIndex; "next" uses currentIndex + 1.
   * Returns an empty string if the resolved index is out of bounds.
   */
  const getMarqueeText = (source: "current" | "next"): string => {
    const index =
      source === "current" ? config.currentIndex : config.currentIndex + 1;
    const item = queue[index];
    return item?.marqueeText || "";
  };

  return (
    <>
      {config.topMarqueeEnabled && (
        <MarqueeBar
          label={config.topMarqueeLabel}
          text={getMarqueeText(config.topMarqueeSource)}
          position="top"
        />
      )}
      {config.bottomMarqueeEnabled && (
        <MarqueeBar
          label={config.bottomMarqueeLabel}
          text={getMarqueeText(config.bottomMarqueeSource)}
          position="bottom"
        />
      )}
    </>
  );
}
