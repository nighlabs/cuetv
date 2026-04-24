import { useEffect, useRef, useCallback } from "react";
import {
  useQuery,
  useQueryClient,
  useMutation,
} from "@tanstack/react-query";
import {
  DndContext,
  closestCenter,
  KeyboardSensor,
  PointerSensor,
  useSensor,
  useSensors,
  type DragEndEvent,
} from "@dnd-kit/core";
import {
  SortableContext,
  sortableKeyboardCoordinates,
  verticalListSortingStrategy,
} from "@dnd-kit/sortable";
import { useSessionStore } from "@/stores/sessionStore";
import { useSocketStore } from "@/stores/socketStore";
import { AuthPortal } from "@/components/AuthPortal";
import { AddVideoForm } from "@/components/AddVideoForm";
import { QueueItemComponent } from "@/components/QueueItem";
import { PlaybackControls } from "@/components/PlaybackControls";
import { RoomConfigPanel } from "@/components/RoomConfigPanel";
import { WebSocketClient } from "@/services/websocketClient";
import { getQueue, getRoomConfig, reorderQueue } from "@/services/api";
import type { WSEvent } from "@/types/events";
import type { QueueItem } from "@/types/api";
import { Copy, Check } from "lucide-react";
import { useState } from "react";

/**
 * Admin remote control page, primarily designed for phone usage.
 * Displays the {@link AuthPortal} when unauthenticated. Once authenticated,
 * shows playback controls, the drag-and-drop queue, video add form, and
 * room configuration panel. Maintains a WebSocket connection for real-time
 * queue and config sync with other admin sessions.
 */
export function AdminPage() {
  const { isAuthenticated, sessionId, friendKey, viewerToken, token } =
    useSessionStore();
  const { wsConnected, setWsConnected } = useSocketStore();
  const queryClient = useQueryClient();
  const wsRef = useRef<WebSocketClient | null>(null);
  const [copied, setCopied] = useState(false);

  const { data: queue = [] } = useQuery({
    queryKey: ["queue", sessionId],
    queryFn: () => getQueue(sessionId!),
    enabled: !!sessionId,
  });

  const { data: config } = useQuery({
    queryKey: ["config", sessionId],
    queryFn: () => getRoomConfig(sessionId!),
    enabled: !!sessionId,
  });

  const reorderMutation = useMutation({
    mutationFn: (order: string[]) => reorderQueue(sessionId!, order),
    onMutate: async (order) => {
      // Cancel outgoing refetches so they don't overwrite optimistic update
      await queryClient.cancelQueries({ queryKey: ["queue", sessionId] });
      const previous = queryClient.getQueryData<QueueItem[]>(["queue", sessionId]);
      // Reorder the cached queue to match the new order
      if (previous) {
        const reordered = order
          .map((id, i) => {
            const item = previous.find((q) => q.id === id);
            return item ? { ...item, position: i } : null;
          })
          .filter(Boolean) as QueueItem[];
        queryClient.setQueryData(["queue", sessionId], reordered);
      }
      return { previous };
    },
    onError: (_err, _order, context) => {
      // Roll back on error
      if (context?.previous) {
        queryClient.setQueryData(["queue", sessionId], context.previous);
      }
    },
    onSettled: () => {
      queryClient.invalidateQueries({ queryKey: ["queue", sessionId] });
    },
  });

  /**
   * Handles incoming WebSocket events by invalidating the relevant TanStack
   * Query caches, which triggers an automatic refetch of the latest data
   * from the backend rather than manually merging state.
   */
  const handleWSEvent = useCallback(
    (event: WSEvent) => {
      if (event.type === "queue:updated") {
        queryClient.invalidateQueries({ queryKey: ["queue", sessionId] });
      } else if (event.type === "config:updated") {
        queryClient.invalidateQueries({ queryKey: ["config", sessionId] });
      }
    },
    [queryClient, sessionId]
  );

  /**
   * WebSocket lifecycle: connects on mount when session credentials are
   * available, passes incoming events to handleWSEvent for cache invalidation,
   * and disconnects on cleanup to avoid leaked connections.
   */
  useEffect(() => {
    if (!sessionId || !token) return;

    const ws = new WebSocketClient(sessionId, token, handleWSEvent, setWsConnected);
    wsRef.current = ws;
    ws.connect();

    return () => {
      ws.disconnect();
      wsRef.current = null;
    };
  }, [sessionId, token, handleWSEvent, setWsConnected]);

  const sensors = useSensors(
    useSensor(PointerSensor),
    useSensor(KeyboardSensor, {
      coordinateGetter: sortableKeyboardCoordinates,
    })
  );

  /**
   * Handles drag-and-drop reorder: removes the dragged item from its old
   * position, inserts it at the new position, then sends the full ordered
   * array of item IDs to the backend via reorderMutation.
   */
  const handleDragEnd = (event: DragEndEvent) => {
    const { active, over } = event;
    if (!over || active.id === over.id) return;

    const oldIndex = queue.findIndex((item) => item.id === active.id);
    const newIndex = queue.findIndex((item) => item.id === over.id);

    const newOrder = [...queue];
    const [moved] = newOrder.splice(oldIndex, 1);
    newOrder.splice(newIndex, 0, moved);

    reorderMutation.mutate(newOrder.map((item) => item.id));
  };

  /**
   * Copies the friend key to the clipboard using the Clipboard API and
   * shows a checkmark icon for 2 seconds as visual feedback before
   * reverting to the copy icon.
   */
  const handleCopyFriendKey = async () => {
    if (friendKey) {
      await navigator.clipboard.writeText(friendKey);
      setCopied(true);
      setTimeout(() => setCopied(false), 2000);
    }
  };

  if (!isAuthenticated) {
    return <AuthPortal />;
  }

  return (
    <div className="min-h-screen bg-zinc-950 text-white">
      <div className="mx-auto max-w-lg space-y-6 p-4">
        {/* Header */}
        <div className="flex items-center justify-between">
          <h1 className="text-xl font-bold">CueTV Admin</h1>
          <div className="flex items-center gap-2">
            <span
              className={`h-2 w-2 rounded-full ${wsConnected ? "bg-green-500" : "bg-red-500"}`}
            />
            <span className="text-xs text-zinc-400">
              {wsConnected ? "Connected" : "Disconnected"}
            </span>
          </div>
        </div>

        {/* Friend Key */}
        {friendKey && (
          <div className="flex items-center justify-between rounded-md border border-zinc-700 bg-zinc-900 p-3">
            <div>
              <p className="text-xs text-zinc-400">Friend Key</p>
              <p className="font-mono text-lg">{friendKey}</p>
            </div>
            <button
              onClick={handleCopyFriendKey}
              className="min-h-[44px] min-w-[44px] rounded p-2 text-zinc-400 hover:text-white"
            >
              {copied ? <Check size={20} /> : <Copy size={20} />}
            </button>
          </div>
        )}

        {/* Viewer URL — includes the viewer token as a query param so the
          viewer page can authenticate SSE connections and fetch queue/config. */}
        {viewerToken && sessionId && (
          <div className="rounded-md border border-zinc-700 bg-zinc-900 p-3">
            <p className="text-xs text-zinc-400">Viewer URL</p>
            <p className="truncate text-sm text-blue-400">
              {window.location.origin}/viewer/{sessionId}?token={viewerToken}
            </p>
          </div>
        )}

        {/* Playback Controls */}
        <PlaybackControls />

        {/* Add Video */}
        <AddVideoForm />

        {/* Queue */}
        <div className="space-y-2">
          <h2 className="text-sm font-medium text-zinc-400">
            Queue ({queue.length})
          </h2>
          <DndContext
            sensors={sensors}
            collisionDetection={closestCenter}
            onDragEnd={handleDragEnd}
          >
            <SortableContext
              items={queue.map((item) => item.id)}
              strategy={verticalListSortingStrategy}
            >
              <div className="space-y-2">
                {queue.map((item) => (
                  <QueueItemComponent
                    key={item.id}
                    item={item}
                    isCurrentlyPlaying={
                      config?.currentIndex === item.position
                    }
                  />
                ))}
              </div>
            </SortableContext>
          </DndContext>
          {queue.length === 0 && (
            <p className="py-8 text-center text-zinc-500">
              No videos in queue. Add one above.
            </p>
          )}
        </div>

        {/* Room Config */}
        <RoomConfigPanel />
      </div>
    </div>
  );
}
