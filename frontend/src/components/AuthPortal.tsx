import { useState } from "react";
import { createSession, joinSession, verifyAdmin } from "@/services/api";
import { useSessionStore } from "@/stores/sessionStore";

/**
 * Three-step authentication portal for the admin interface.
 *
 * **Auth flow:**
 * 1. **"password"** — User enters the admin portal password, which is verified
 *    against the backend without creating a session.
 * 2. **"choose"** — User decides whether to create a brand-new session or join
 *    an existing one via a friend key.
 * 3. **"join"** — (Only if joining) User enters an `adjective-noun-number`
 *    friend key to join another admin's session with a friend-role JWT.
 *
 * On successful session creation or join, the JWT and session details are stored
 * in the Zustand session store (in memory, not localStorage). The plaintext
 * password is cleared from component state immediately after use to limit its
 * lifetime in memory.
 */
export function AuthPortal() {
  /**
   * Controls which panel is rendered. Transitions:
   * - "password" → "choose"  (on successful password verification)
   * - "choose"  → "join"     (user clicks "Join Session")
   * - "join"    → "choose"   (user clicks "Back")
   * Creating a session from "choose" exits the portal entirely (session store
   * is populated, so AdminPage renders the dashboard instead).
   */
  const [step, setStep] = useState<"password" | "choose" | "join">("password");
  const [password, setPassword] = useState("");
  const [friendKey, setFriendKey] = useState("");
  const [error, setError] = useState("");
  const [loading, setLoading] = useState(false);
  const setSession = useSessionStore((s) => s.setSession);

  /**
   * Verifies the admin portal password against the backend via
   * `POST /api/admin/verify`. This step gates access to session management
   * without creating a session or issuing a JWT — it only confirms the user
   * knows the password. On success, advances to the "choose" step.
   */
  const handlePasswordSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setError("");
    setLoading(true);

    try {
      const result = await verifyAdmin(password);
      if (result.valid) {
        setStep("choose");
      } else {
        setError("Invalid password");
      }
    } catch {
      setError("Failed to verify password");
    } finally {
      setLoading(false);
    }
  };

  /**
   * Creates a new session via `POST /api/sessions`, authenticating with the
   * password collected in step 1. On success, stores the admin JWT, session ID,
   * friend key, and viewer token in the Zustand session store. The plaintext
   * password is cleared from state immediately after the session is created to
   * minimize its lifetime in memory.
   */
  const handleCreateSession = async () => {
    setError("");
    setLoading(true);

    try {
      const result = await createSession(password);
      setPassword("");
      setSession({
        token: result.token,
        sessionId: result.sessionId,
        friendKey: result.friendKey,
        viewerToken: result.viewerToken,
      });
    } catch {
      setError("Failed to create session");
    } finally {
      setLoading(false);
    }
  };

  /**
   * Joins an existing session via `POST /api/sessions/join` using a
   * human-readable friend key (e.g. "happy-tiger-42"). The backend returns a
   * friend-role JWT with limited permissions compared to the session creator.
   * The viewer token is set to an empty string since friends do not manage the
   * viewer display directly.
   */
  const handleJoinSession = async (e: React.FormEvent) => {
    e.preventDefault();
    setError("");
    setLoading(true);

    try {
      const result = await joinSession(friendKey);
      setSession({
        token: result.token,
        sessionId: result.sessionId,
        friendKey: friendKey,
        viewerToken: "",
      });
    } catch {
      setError("Invalid friend key");
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="flex min-h-screen items-center justify-center bg-zinc-950 p-4">
      <div className="w-full max-w-sm space-y-6">
        <h1 className="text-center text-3xl font-bold text-white">CueTV</h1>

        {error && (
          <div className="rounded-md bg-red-900/50 p-3 text-sm text-red-300">
            {error}
          </div>
        )}

        {step === "password" && (
          <form onSubmit={handlePasswordSubmit} className="space-y-4">
            <input
              type="password"
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              placeholder="Admin password"
              className="w-full rounded-md border border-zinc-700 bg-zinc-800 px-4 py-3 text-white placeholder-zinc-500 focus:border-blue-500 focus:outline-none"
              autoFocus
            />
            <button
              type="submit"
              disabled={loading || !password}
              className="w-full rounded-md bg-blue-600 px-4 py-3 font-medium text-white hover:bg-blue-700 disabled:opacity-50"
            >
              {loading ? "Verifying..." : "Sign In"}
            </button>
          </form>
        )}

        {step === "choose" && (
          <div className="space-y-3">
            <button
              onClick={handleCreateSession}
              disabled={loading}
              className="w-full rounded-md bg-blue-600 px-4 py-3 font-medium text-white hover:bg-blue-700 disabled:opacity-50"
            >
              {loading ? "Creating..." : "New Session"}
            </button>
            <button
              onClick={() => setStep("join")}
              className="w-full rounded-md border border-zinc-700 px-4 py-3 font-medium text-white hover:bg-zinc-800"
            >
              Join Session
            </button>
          </div>
        )}

        {step === "join" && (
          <form onSubmit={handleJoinSession} className="space-y-4">
            <input
              type="text"
              value={friendKey}
              onChange={(e) => setFriendKey(e.target.value)}
              placeholder="Friend key (e.g. happy-tiger-42)"
              className="w-full rounded-md border border-zinc-700 bg-zinc-800 px-4 py-3 text-white placeholder-zinc-500 focus:border-blue-500 focus:outline-none"
              autoFocus
            />
            <button
              type="submit"
              disabled={loading || !friendKey}
              className="w-full rounded-md bg-blue-600 px-4 py-3 font-medium text-white hover:bg-blue-700 disabled:opacity-50"
            >
              {loading ? "Joining..." : "Join"}
            </button>
            <button
              type="button"
              onClick={() => setStep("choose")}
              className="w-full rounded-md border border-zinc-700 px-4 py-3 font-medium text-zinc-400 hover:bg-zinc-800"
            >
              Back
            </button>
          </form>
        )}
      </div>
    </div>
  );
}
