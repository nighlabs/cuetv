import { useState, useEffect } from "react";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { getRoomConfig, updateRoomConfig } from "@/services/api";
import { useSessionStore } from "@/stores/sessionStore";
import type { UpdateRoomConfigRequest } from "@/types/api";

/**
 * Controls room marquee configuration (top and bottom marquee bars).
 *
 * Each setting is sent as a partial PATCH to `/api/sessions/:id/config` using
 * pointer fields in `UpdateRoomConfigRequest` so the backend can distinguish
 * "not provided" from "set to zero-value". Only changed fields are included
 * in each request.
 */
export function RoomConfigPanel() {
  const sessionId = useSessionStore((s) => s.sessionId);
  const queryClient = useQueryClient();

  const { data: config } = useQuery({
    queryKey: ["config", sessionId],
    queryFn: () => getRoomConfig(sessionId!),
    enabled: !!sessionId,
  });

  const mutation = useMutation({
    mutationFn: (update: UpdateRoomConfigRequest) =>
      updateRoomConfig(sessionId!, update),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["config", sessionId] });
    },
  });

  /**
   * Local state for label inputs. We buffer keystrokes here instead of
   * calling `mutation.mutate()` on every `onChange`, which would send a
   * PATCH request per character typed. The actual mutation fires on blur
   * only when the value differs from the server state.
   */
  const [topLabel, setTopLabel] = useState(config?.topMarqueeLabel ?? "");
  const [bottomLabel, setBottomLabel] = useState(
    config?.bottomMarqueeLabel ?? "",
  );

  // Sync local label state when config changes from the server (e.g. another
  // admin updates it via WebSocket, triggering a TanStack Query cache
  // invalidation and refetch).
  useEffect(() => {
    if (config) {
      setTopLabel(config.topMarqueeLabel);
      setBottomLabel(config.bottomMarqueeLabel);
    }
  }, [config?.topMarqueeLabel, config?.bottomMarqueeLabel]);

  if (!config) return null;

  const labelClass = "text-sm text-zinc-400";
  const inputClass =
    "w-full rounded-md border border-zinc-700 bg-zinc-800 px-3 py-2 text-sm text-white focus:border-blue-500 focus:outline-none";
  const selectClass =
    "w-full rounded-md border border-zinc-700 bg-zinc-800 px-3 py-2 text-sm text-white focus:border-blue-500 focus:outline-none";

  return (
    <div className="space-y-4 rounded-md border border-zinc-700 bg-zinc-900 p-4">
      <h3 className="font-medium text-white">Room Config</h3>

      {/* Top Marquee */}
      <div className="space-y-2">
        <div className="flex items-center justify-between">
          <label className={labelClass}>Top Marquee</label>
          {/*
           * Toggle switch — inline custom implementation matching shadcn/ui
           * Switch styling. Fires a PATCH immediately on click since it is a
           * single discrete state change (not continuous input like text).
           */}
          <button
            onClick={() =>
              mutation.mutate({
                topMarqueeEnabled: !config.topMarqueeEnabled,
              })
            }
            className={`relative inline-flex h-6 w-11 items-center rounded-full transition-colors ${
              config.topMarqueeEnabled ? "bg-blue-600" : "bg-zinc-600"
            }`}
          >
            <span
              className={`inline-block h-4 w-4 transform rounded-full bg-white transition-transform ${
                config.topMarqueeEnabled ? "translate-x-6" : "translate-x-1"
              }`}
            />
          </button>
        </div>
        {config.topMarqueeEnabled && (
          <>
            <input
              type="text"
              value={topLabel}
              onChange={(e) => setTopLabel(e.target.value)}
              onBlur={() => {
                if (topLabel !== config.topMarqueeLabel) {
                  mutation.mutate({ topMarqueeLabel: topLabel });
                }
              }}
              placeholder="Label (e.g. Now Playing)"
              maxLength={50}
              className={inputClass}
            />
            {/*
             * Source selector — controls whether the marquee displays the
             * current video's text or the next video's text. Fires a PATCH
             * immediately on change since it is a discrete selection.
             */}
            <select
              value={config.topMarqueeSource}
              onChange={(e) =>
                mutation.mutate({
                  topMarqueeSource: e.target.value as "current" | "next",
                })
              }
              className={selectClass}
            >
              <option value="current">Current Video</option>
              <option value="next">Next Video</option>
            </select>
          </>
        )}
      </div>

      {/* Bottom Marquee */}
      <div className="space-y-2">
        <div className="flex items-center justify-between">
          <label className={labelClass}>Bottom Marquee</label>
          {/*
           * Toggle switch — inline custom implementation matching shadcn/ui
           * Switch styling. Fires a PATCH immediately on click since it is a
           * single discrete state change (not continuous input like text).
           */}
          <button
            onClick={() =>
              mutation.mutate({
                bottomMarqueeEnabled: !config.bottomMarqueeEnabled,
              })
            }
            className={`relative inline-flex h-6 w-11 items-center rounded-full transition-colors ${
              config.bottomMarqueeEnabled ? "bg-blue-600" : "bg-zinc-600"
            }`}
          >
            <span
              className={`inline-block h-4 w-4 transform rounded-full bg-white transition-transform ${
                config.bottomMarqueeEnabled ? "translate-x-6" : "translate-x-1"
              }`}
            />
          </button>
        </div>
        {config.bottomMarqueeEnabled && (
          <>
            <input
              type="text"
              value={bottomLabel}
              onChange={(e) => setBottomLabel(e.target.value)}
              onBlur={() => {
                if (bottomLabel !== config.bottomMarqueeLabel) {
                  mutation.mutate({ bottomMarqueeLabel: bottomLabel });
                }
              }}
              placeholder="Label (e.g. Up Next)"
              maxLength={50}
              className={inputClass}
            />
            {/*
             * Source selector — controls whether the marquee displays the
             * current video's text or the next video's text. Fires a PATCH
             * immediately on change since it is a discrete selection.
             */}
            <select
              value={config.bottomMarqueeSource}
              onChange={(e) =>
                mutation.mutate({
                  bottomMarqueeSource: e.target.value as "current" | "next",
                })
              }
              className={selectClass}
            >
              <option value="current">Current Video</option>
              <option value="next">Next Video</option>
            </select>
          </>
        )}
      </div>
    </div>
  );
}
