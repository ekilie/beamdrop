/**
 * Reads the CSRF token from the beamdrop_csrf cookie.
 */
function getCSRFToken(): string | null {
  const match = document.cookie
    .split("; ")
    .find((row) => row.startsWith("beamdrop_csrf="));
  return match ? match.split("=")[1] : null;
}

const UNSAFE_METHODS = new Set(["POST", "PUT", "DELETE", "PATCH"]);

/**
 * Installs a global fetch interceptor that automatically attaches
 * the X-CSRF-Token header to all state-changing requests.
 * Call once at application startup.
 */
export function installCSRFInterceptor(): void {
  const originalFetch = window.fetch.bind(window);

  window.fetch = async (
    input: RequestInfo | URL,
    init?: RequestInit,
  ): Promise<Response> => {
    const method = (init?.method || "GET").toUpperCase();

    if (UNSAFE_METHODS.has(method)) {
      const token = getCSRFToken();
      if (token) {
        const headers = new Headers(init?.headers);
        if (!headers.has("X-CSRF-Token")) {
          headers.set("X-CSRF-Token", token);
        }
        init = { ...init, headers };
      }
    }

    return originalFetch(input, init);
  };
}

/**
 * A fetch wrapper that automatically includes the CSRF token header
 * for state-changing requests (POST, PUT, DELETE, PATCH).
 */
export async function safeFetch(
  input: RequestInfo | URL,
  init?: RequestInit,
): Promise<Response> {
  const method = (init?.method || "GET").toUpperCase();

  if (UNSAFE_METHODS.has(method)) {
    const token = getCSRFToken();
    if (token) {
      const headers = new Headers(init?.headers);
      headers.set("X-CSRF-Token", token);
      init = { ...init, headers };
    }
  }

  return fetch(input, init);
}
