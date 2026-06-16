import { Component, type ErrorInfo, type ReactNode } from "react";

interface Props {
  children: ReactNode;
}

interface State {
  error: Error | null;
}

export class ErrorBoundary extends Component<Props, State> {
  state: State = { error: null };

  static getDerivedStateFromError(error: Error): State {
    return { error };
  }

  componentDidCatch(error: Error, info: ErrorInfo) {
    console.error("Aide UI crashed:", error, info);
  }

  reset = () => this.setState({ error: null });

  render() {
    if (!this.state.error) return this.props.children;

    return (
      <div className="flex h-screen flex-col items-center justify-center gap-4 bg-background p-6 text-center">
        <div className="max-w-md space-y-2">
          <h1 className="text-lg font-semibold">Something went wrong</h1>
          <p className="text-sm text-muted-foreground">
            The interface hit an unexpected error. You can try reloading.
          </p>
          <pre className="max-h-40 overflow-auto rounded-md bg-muted p-3 text-left text-xs text-muted-foreground">
            {this.state.error.message}
          </pre>
        </div>
        <div className="flex items-center gap-2">
          <button
            onClick={this.reset}
            className="rounded-md border border-input bg-background px-4 py-2 text-sm font-medium transition-colors hover:bg-accent"
          >
            Try again
          </button>
          <button
            onClick={() => window.location.reload()}
            className="rounded-md bg-primary px-4 py-2 text-sm font-medium text-primary-foreground transition-colors hover:bg-primary/90"
          >
            Reload
          </button>
        </div>
      </div>
    );
  }
}
