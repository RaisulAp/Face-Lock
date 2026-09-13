// Cross-tab synchronization via BroadcastChannel.
// Important (D23): Never broadcast raw tokens over the channel.
// Only broadcast synchronization events (timestamps, user IDs).

export type AuthEvent =
  | { type: "auth:refreshed"; timestamp: number }
  | { type: "auth:logout" }
  | { type: "auth:user-changed"; userId: string };

type Listener = (event: AuthEvent) => void;

const CHANNEL_NAME = "faceclock-auth";
let channel: BroadcastChannel | null = null;
const listeners = new Set<Listener>();

if (typeof window !== "undefined" && "BroadcastChannel" in window) {
  try {
    channel = new BroadcastChannel(CHANNEL_NAME);
    channel.onmessage = (e: MessageEvent<AuthEvent>) => {
      if (e.data && typeof e.data.type === "string") {
        listeners.forEach((listener) => {
          try {
            listener(e.data);
          } catch (err) {
            console.error("Auth broadcast listener error:", err);
          }
        });
      }
    };
  } catch (err) {
    console.warn("BroadcastChannel not supported or blocked:", err);
  }
}

export const authBroadcast = {
  send(event: AuthEvent): void {
    if (channel) {
      try {
        channel.postMessage(event);
      } catch (err) {
        console.warn("Failed to post message to BroadcastChannel:", err);
      }
    }
  },

  subscribe(listener: Listener): () => void {
    listeners.add(listener);
    return () => {
      listeners.delete(listener);
    };
  },
};
