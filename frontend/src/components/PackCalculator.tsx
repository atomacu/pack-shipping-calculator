import { FormEvent, useEffect, useRef, useState } from "react";

import {
  PackingPlan,
  calculateOrder,
  fetchPackSizes,
  replacePackSizes
} from "../api/client";

type SettingsNotice = {
  kind: "saving" | "success" | "error";
  message: string;
};

export function PackCalculator() {
  const [packSizes, setPackSizes] = useState<number[]>([]);
  const [packSizesDraft, setPackSizesDraft] = useState("");
  const [items, setItems] = useState("");
  const [plan, setPlan] = useState<PackingPlan | null>(null);
  const [error, setError] = useState("");
  const [settingsNotice, setSettingsNotice] = useState<SettingsNotice | null>(null);
  const [isLoading, setIsLoading] = useState(false);
  const [hasLoadedPackSizes, setHasLoadedPackSizes] = useState(false);
  const latestSaveId = useRef(0);
  const savedPackSizesDraft = useRef("");

  useEffect(() => {
    fetchPackSizes()
      .then((sizes) => {
        const formatted = formatPackSizes(sizes);
        setPackSizes(sizes);
        setPackSizesDraft(formatted);
        savedPackSizesDraft.current = formatted;
        setHasLoadedPackSizes(true);
      })
      .catch((err: Error) => setError(err.message));
  }, []);

  useEffect(() => {
    if (!hasLoadedPackSizes || packSizesDraft.trim() === savedPackSizesDraft.current) {
      return;
    }

    const saveId = latestSaveId.current + 1;
    latestSaveId.current = saveId;
    const parsedPackSizes = parsePackSizes(packSizesDraft);
    if (!parsedPackSizes) {
      setSettingsNotice({
        kind: "error",
        message: "Autosave paused: enter positive whole-number pack sizes."
      });
      return;
    }

    const normalizedPackSizes = normalizePackSizes(parsedPackSizes);
    setSettingsNotice({ kind: "saving", message: "Saving pack sizes..." });

    const timeout = window.setTimeout(async () => {
      try {
        if (arraysEqual(normalizedPackSizes, packSizes)) {
          const formatted = formatPackSizes(normalizedPackSizes);
          savedPackSizesDraft.current = formatted;
          setPackSizesDraft(formatted);
          setSettingsNotice({ kind: "success", message: "Pack sizes saved automatically." });
          return;
        }

        const saved = await replacePackSizes(normalizedPackSizes);
        if (saveId !== latestSaveId.current) {
          return;
        }

        const formatted = formatPackSizes(saved);
        setPackSizes(saved);
        setPackSizesDraft(formatted);
        savedPackSizesDraft.current = formatted;
        setPlan(null);
        setSettingsNotice({ kind: "success", message: "Pack sizes saved automatically." });
      } catch (err) {
        if (saveId !== latestSaveId.current) {
          return;
        }

        setSettingsNotice({
          kind: "error",
          message: err instanceof Error ? err.message : "Could not save pack sizes"
        });
      }
    }, 700);

    return () => window.clearTimeout(timeout);
  }, [hasLoadedPackSizes, packSizes, packSizesDraft]);

  async function onSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setError("");
    setPlan(null);

    const parsedItems = Number(items);
    if (!Number.isInteger(parsedItems) || parsedItems <= 0) {
      setError("Enter a whole number greater than 0.");
      return;
    }

    setIsLoading(true);
    try {
      setPlan(await calculateOrder(parsedItems));
    } catch (err) {
      setError(err instanceof Error ? err.message : "Calculation failed");
    } finally {
      setIsLoading(false);
    }
  }

  return (
    <section className="calculator">
      <div className="calculator__header">
        <h1>Pack Shipping Calculator</h1>
        <div className="pack-list" aria-label="Configured pack sizes">
          {packSizes.map((size) => (
            <span className="pack-chip" key={size}>
              {size}
            </span>
          ))}
        </div>
      </div>

      <div className="calculator__form settings-form">
        <label htmlFor="pack-sizes">Pack sizes</label>
        <div className="input-row input-row--single">
          <input
            id="pack-sizes"
            onChange={(event) => setPackSizesDraft(event.target.value)}
            placeholder="250, 500, 1000, 2000, 5000"
            type="text"
            value={packSizesDraft}
          />
        </div>

        {settingsNotice ? (
          <p
            aria-live="polite"
            className={`message message--${settingsNotice.kind}`}
            role={settingsNotice.kind === "error" ? "alert" : "status"}
          >
            {settingsNotice.message}
          </p>
        ) : null}
      </div>

      <form className="calculator__form" onSubmit={onSubmit}>
        <label htmlFor="items">Items ordered</label>
        <div className="input-row">
          <input
            id="items"
            inputMode="numeric"
            min="1"
            onChange={(event) => setItems(event.target.value)}
            placeholder="12001"
            type="number"
            value={items}
          />
          <button disabled={isLoading} type="submit">
            {isLoading ? "Calculating" : "Calculate"}
          </button>
        </div>
      </form>

      {error ? <p className="message message--error">{error}</p> : null}

      {plan ? (
        <section className="result" aria-label="Packing result">
          <dl className="summary-grid">
            <div>
              <dt>Items shipped</dt>
              <dd>{plan.items_shipped}</dd>
            </div>
            <div>
              <dt>Items over</dt>
              <dd>{plan.items_over}</dd>
            </div>
            <div>
              <dt>Total packs</dt>
              <dd>{plan.total_packs}</dd>
            </div>
          </dl>

          <table>
            <thead>
              <tr>
                <th>Pack size</th>
                <th>Quantity</th>
              </tr>
            </thead>
            <tbody>
              {plan.packs.map((pack) => (
                <tr key={pack.size}>
                  <td>{pack.size}</td>
                  <td>{pack.quantity}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </section>
      ) : null}
    </section>
  );
}

function formatPackSizes(packSizes: number[]) {
  return packSizes.join(", ");
}

function parsePackSizes(value: string) {
  const parts = value
    .split(/[,\s]+/)
    .map((part) => part.trim())
    .filter(Boolean);

  if (parts.length === 0) {
    return null;
  }

  const sizes = parts.map(Number);
  if (sizes.some((size) => !Number.isInteger(size) || size <= 0)) {
    return null;
  }

  return sizes;
}

function normalizePackSizes(packSizes: number[]) {
  return [...new Set(packSizes)].sort((a, b) => a - b);
}

function arraysEqual(left: number[], right: number[]) {
  return left.length === right.length && left.every((value, index) => value === right[index]);
}
