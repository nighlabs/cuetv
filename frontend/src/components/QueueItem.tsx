import { memo } from "react";
import { useSortable } from "@dnd-kit/sortable";
import { CSS } from "@dnd-kit/utilities";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { deleteQueueItem } from "@/services/api";
import { getThumbnailUrl } from "@/services/youtubeValidation";
import { useSessionStore } from "@/stores/sessionStore";
import type { QueueItem as QueueItemType } from "@/types/api";
import { GripVertical, Trash2 } from "lucide-react";

/** Props for {@link QueueItemComponent}. */
interface QueueItemProps {
  item: QueueItemType;
  /** Controls whether this item receives the visual highlight (blue border
   *  and "Now Playing" badge) indicating it is the active video. */
  isCurrentlyPlaying: boolean;
}

/**
 * A single queue item row. Wrapped in React.memo to prevent unnecessary
 * re-renders during drag-and-drop operations, since only the dragged item
 * and its new neighbors actually change.
 */
export const QueueItemComponent = memo(function QueueItemComponent({
  item,
  isCurrentlyPlaying,
}: QueueItemProps) {
  const sessionId = useSessionStore((s) => s.sessionId);
  const queryClient = useQueryClient();

  const deleteMutation = useMutation({
    mutationFn: () => deleteQueueItem(sessionId!, item.id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["queue", sessionId] });
    },
  });

  // useSortable provides the ref, drag listeners, and transform/transition
  // values needed to make this item a draggable target within the
  // SortableContext. isDragging is used to reduce opacity while dragging.
  const {
    attributes,
    listeners,
    setNodeRef,
    transform,
    transition,
    isDragging,
  } = useSortable({ id: item.id });

  const style = {
    transform: CSS.Transform.toString(transform),
    transition,
  };

  return (
    <div
      ref={setNodeRef}
      style={style}
      className={`flex items-center gap-3 rounded-md border p-3 ${
        isCurrentlyPlaying
          ? "border-blue-500 bg-blue-950/30"
          : "border-zinc-700 bg-zinc-800"
      } ${isDragging ? "opacity-50" : ""}`}
    >
      <button
        {...attributes}
        {...listeners}
        className="min-h-[44px] min-w-[44px] cursor-grab touch-none rounded p-2 text-zinc-400 hover:text-white active:cursor-grabbing"
      >
        <GripVertical size={20} />
      </button>

      <img
        src={getThumbnailUrl(item.youtubeVideoId)}
        alt=""
        className="h-12 w-16 rounded object-cover"
      />

      <div className="min-w-0 flex-1">
        <p className="truncate text-sm text-zinc-300">
          {item.marqueeText || item.youtubeVideoId}
        </p>
        {isCurrentlyPlaying && (
          <p className="text-xs text-blue-400">Now Playing</p>
        )}
      </div>

      <button
        onClick={() => deleteMutation.mutate()}
        disabled={deleteMutation.isPending}
        className="min-h-[44px] min-w-[44px] rounded p-2 text-zinc-400 hover:text-red-400 disabled:opacity-50"
      >
        <Trash2 size={20} />
      </button>
    </div>
  );
});
