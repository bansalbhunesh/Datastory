import { Component, type ReactNode } from "react";

type Props = { children: ReactNode };
type State = { error: Error | null };

export class ErrorBoundary extends Component<Props, State> {
  state: State = { error: null };

  static getDerivedStateFromError(error: Error): State {
    return { error };
  }

  componentDidCatch(error: Error, info: { componentStack?: string }): void {
    // Surfaced to the dev console; production users see the fallback UI.
    console.error("UI crash:", error, info.componentStack);
  }

  reset = () => this.setState({ error: null });

  render() {
    if (this.state.error) {
      return (
        <div className="mx-auto mt-16 max-w-xl rounded-2xl border border-rose-500/30 bg-rose-500/10 p-6 text-sm text-rose-100">
          <div className="text-base font-semibold text-rose-200">Something went wrong rendering the page.</div>
          <pre className="mt-3 max-h-64 overflow-auto whitespace-pre-wrap break-words rounded-lg bg-slate-950/60 p-3 text-xs text-rose-100/90">
            {this.state.error.message}
          </pre>
          <button
            type="button"
            onClick={this.reset}
            className="mt-4 rounded-lg border border-rose-400/30 px-3 py-1.5 text-xs font-semibold text-rose-100 hover:border-rose-300"
          >
            Try again
          </button>
        </div>
      );
    }
    return this.props.children;
  }
}
