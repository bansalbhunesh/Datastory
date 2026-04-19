type Props = {
  value: string;
  onChange: (next: string) => void;
  onSubmit: () => void;
  disabled?: boolean;
  placeholder?: string;
};

export function SearchBar({ value, onChange, onSubmit, disabled, placeholder }: Props) {
  return (
    <div className="flex flex-col gap-2 sm:flex-row sm:items-center">
      <input
        value={value}
        disabled={disabled}
        onChange={(e) => onChange(e.target.value)}
        onKeyDown={(e) => {
          if (e.key === "Enter") onSubmit();
        }}
        placeholder={placeholder ?? "Search tables (e.g. dim_address) or paste a FQN"}
        className="w-full rounded-lg border border-slate-800 bg-slate-900 px-3 py-2 text-sm text-slate-100 outline-none ring-0 placeholder:text-slate-500 focus:border-indigo-400"
      />
      <button
        type="button"
        disabled={disabled}
        onClick={onSubmit}
        className="rounded-lg bg-indigo-500 px-4 py-2 text-sm font-semibold text-white shadow hover:bg-indigo-400 disabled:cursor-not-allowed disabled:opacity-50"
      >
        Generate report
      </button>
    </div>
  );
}
