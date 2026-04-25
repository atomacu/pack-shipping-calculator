export type PackLine = {
  size: number;
  quantity: number;
};

export type PackingPlan = {
  items_ordered: number;
  items_shipped: number;
  items_over: number;
  total_packs: number;
  packs: PackLine[];
};

type PacksResponse = {
  pack_sizes: number[];
};

type ErrorResponse = {
  error?: {
    message?: string;
  };
};

const API_BASE = (import.meta.env.VITE_API_BASE ?? "").replace(/\/$/, "");

export async function fetchPackSizes(): Promise<number[]> {
  const response = await fetch(apiURL("/api/v1/packs"));
  const payload = await readJSON<PacksResponse>(response, "Could not load pack sizes");
  return payload.pack_sizes;
}

export async function replacePackSizes(packSizes: number[]): Promise<number[]> {
  const response = await fetch(apiURL("/api/v1/packs"), {
    method: "PUT",
    headers: {
      "Content-Type": "application/json"
    },
    body: JSON.stringify({ pack_sizes: packSizes })
  });

  const payload = await readJSON<PacksResponse>(response, "Could not save pack sizes");
  return payload.pack_sizes;
}

export async function calculateOrder(items: number): Promise<PackingPlan> {
  const response = await fetch(apiURL("/api/v1/orders/calculate"), {
    method: "POST",
    headers: {
      "Content-Type": "application/json"
    },
    body: JSON.stringify({ items })
  });

  return readJSON<PackingPlan>(response, "Could not calculate order");
}

function apiURL(path: string): string {
  return `${API_BASE}${path}`;
}

async function readJSON<T>(response: Response, fallbackMessage: string): Promise<T> {
  const text = await response.text();
  let payload: unknown = null;

  if (text) {
    try {
      payload = JSON.parse(text);
    } catch {
      throw new Error(
        response.ok
          ? "The server returned an invalid response."
          : `${fallbackMessage}: server returned HTTP ${response.status}.`
      );
    }
  }

  if (!response.ok) {
    throw new Error(errorMessage(payload) ?? fallbackMessage);
  }

  return payload as T;
}

function errorMessage(payload: unknown) {
  if (!payload || typeof payload !== "object") {
    return null;
  }

  const error = (payload as ErrorResponse).error;
  return typeof error?.message === "string" ? error.message : null;
}
