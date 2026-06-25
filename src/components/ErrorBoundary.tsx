import { Component, type ReactNode } from "react";

// Contains a render-time crash in one tab instead of blanking the whole app.
export class ErrorBoundary extends Component<
  { children: ReactNode },
  { error: Error | null }
> {
  state = { error: null as Error | null };

  static getDerivedStateFromError(error: Error) {
    return { error };
  }

  render() {
    if (this.state.error) {
      return (
        <div className="error-banner">
          Something went wrong rendering this view — {this.state.error.message}
          <button
            type="button"
            className="btn"
            style={{ marginLeft: 12 }}
            onClick={() => this.setState({ error: null })}
          >
            Try again
          </button>
        </div>
      );
    }
    return this.props.children;
  }
}
