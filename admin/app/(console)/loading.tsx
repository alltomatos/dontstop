export default function ConsoleLoading() {
  return (
    <div className="stack" aria-busy="true" aria-live="polite">
      <div className="skeleton skeleton-title" />
      <div className="grid-3">
        <div className="skeleton skeleton-card" />
        <div className="skeleton skeleton-card" />
        <div className="skeleton skeleton-card" />
      </div>
      <div className="skeleton skeleton-table" />
    </div>
  );
}
