import { useEffect, useId, useRef, useState } from "react";
import { searchTables, type TableSearchHit } from "../api/search";
import { useDebounced } from "../hooks/useDebounced";

type Props = {
  value: string;
  onChange: (next: string) => void;
  onSubmit: () => void;
  disabled?: boolean;
  mockMode: boolean;
  placeholder?: string;
};

export function TableQueryField({ value, onChange, onSubmit, disabled, mockMode, placeholder }: Props) {
  const listId = useId();
  const wrapRef = useRef<HTMLDivElement>(null);
  const [open, setOpen] = useState(false);
  const [hits, setHits] = useState<TableSearchHit[]>([]);
  const [loading, setLoading] = useState(false);
  const debounced = useDebounced(value.trim(), 280);

  useEffect(() => {
    if (mockMode || debounced.length < 2) {
      setHits([]);
      return;
    }
    let cancelled = false;
    setLoading(true);
    searchTables(debounced)
      .then((h) => {
        if (!cancelled) setHits(h);
      })
      .catch(() => {
        if (!cancelled) setHits([]);
      })
      .finally(() => {
        if (!cancelled) setLoading(false);
      });
    return () => {
      cancelled = true;
    };
  }, [debounced, mockMode]);

  useEffect(() => {
    function onDocMouseDown(e: MouseEvent) {
      if (!wrapRef.current) return;
      if (!wrapRef.current.contains(e.target as Node)) setOpen(false);
    }
    document.addEventListener("mousedown", onDocMouseDown);
    return () => document.removeEventListener("mousedown", onDocMouseDown);
  }, []);

  return (
    <div ref={wrapRef} className="relative flex flex-col gap-2 sm:flex-row sm:items-center">
      <div className="relative w-full">
        <input
          value={value}
          disabled={disabled}
          onFocus={() => setOpen(true)}
          onChange={(e) => {
            onChange(e.target.value);
            setOpen(true);
          }}
          onKeyDown={(e) => {
            if (e.key === "Enter") {
              setOpen(false);
              onSubmit();
            }
            if (e.key === "Escape") setOpen(false);
          }}
          placeholder={placeholder ?? "Search tables (e.g. dim_address) or paste a full FQN"}
          className="w-full rounded-lg border border-slate-800 bg-slate-900 px-3 py-2 text-sm text-slate-100 outline-none placeholder:text-slate-500 focus:border-indigo-400"
          aria-autocomplete="list"
          aria-controls={listId}
        />
        {!mockMode && open && debounced.length >= 2 ? (
          <div
            id={listId}
            role="listbox"
            className="absolute z-20 mt-1 max-h-64 w-full overflow-auto rounded-lg border border-slate-800 bg-slate-950 py-1 shadow-xl"
          >
            {loading ? (
              <div className="px-3 py-2 text-xs text-slate-500">Searching OpenMetadata…</div>
            ) : hits.length === 0 ? (
              <div className="px-3 py-2 text-xs text-slate-500">No matches</div>
            ) : (
              hits.map((h) => (
                <button
                  key={h.id || h.fullyQualifiedName}
                  type="button"
                  role="option"
                  className="flex w-full flex-col items-start gap-0.5 px-3 py-2 text-left text-sm hover:bg-slate-900"
                  onMouseDown={(e) => e.preventDefault()}
                  onClick={() => {
                    onChange(h.fullyQualifiedName);
                    setOpen(false);
                  }}
                >
                  <span className="font-medium text-slate-100">{h.name}</span>
                  <span className="break-all text-xs text-slate-500">{h.fullyQualifiedName}</span>
                </button>
              ))
            )}
          </div>
        ) : null}
      </div>
      <button
        type="button"
        disabled={disabled}
        onClick={() => {
          setOpen(false);
          onSubmit();
        }}
        className="rounded-lg bg-indigo-500 px-4 py-2 text-sm font-semibold text-white shadow hover:bg-indigo-400 disabled:cursor-not-allowed disabled:opacity-50"
      >
        Generate report
      </button>
    </div>
  );
}
