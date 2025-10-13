import React, { useState } from 'react';

type Props = {
  onCreate: (name: string, isPrivate: boolean) => Promise<void> | void;
  onCancel?: () => void;
};

export default function ChannelCreateForm({ onCreate, onCancel }: Props) {
  const [name, setName] = useState('');
  const [isPrivate, setIsPrivate] = useState(false);
  const [busy, setBusy] = useState(false);

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    if (!name.trim() || busy) return;
    setBusy(true);
    try {
      await onCreate(name.trim(), isPrivate);
      setName('');
      setIsPrivate(false);
    } finally {
      setBusy(false);
    }
  }

  return (
    <form
      onSubmit={handleSubmit}
      style={{
        border: '1px solid #e5e7eb',
        borderRadius: 8,
        padding: 12,
        marginTop: 8,
        display: 'grid',
        gap: 8,
      }}
    >
      <label style={{ display: 'grid', gap: 4 }}>
        <span style={{ fontSize: 12, color: '#6b7280' }}>Channel name</span>
        <input
          value={name}
          onChange={(e) => setName(e.target.value)}
          placeholder="general"
          disabled={busy}
        />
      </label>

      <label style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
        <input
          type="checkbox"
          checked={isPrivate}
          onChange={(e) => setIsPrivate(e.target.checked)}
          disabled={busy}
        />
        <span>Make this channel private</span>
      </label>

      <div style={{ display: 'flex', gap: 8, justifyContent: 'flex-end' }}>
        {onCancel && (
          <button type="button" className="btn" onClick={onCancel} disabled={busy}>
            Cancel
          </button>
        )}
        <button
          type="submit"
          className="btn"
          disabled={busy || !name.trim()}
          title={!name.trim() ? 'Enter a name' : ''}
        >
          {busy ? 'Creating...' : 'Create'}
        </button>
      </div>
    </form>
  );
}
