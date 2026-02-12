'use client';

interface PaginationProps {
  page: number;
  pageSize: number;
  total: number;
  onChange: (page: number) => void;
}

export default function Pagination({ page, pageSize, total, onChange }: PaginationProps) {
  const totalPages = Math.max(1, Math.ceil(total / pageSize));
  const from = (page - 1) * pageSize + 1;
  const to = Math.min(page * pageSize, total);

  const pages: number[] = [];
  const start = Math.max(1, page - 2);
  const end = Math.min(totalPages, page + 2);
  for (let i = start; i <= end; i++) pages.push(i);

  return (
    <div className="flex items-center justify-between text-sm mt-4">
      <span className="text-black/50">
        {total > 0 ? `${from}–${to} of ${total}` : 'No results'}
      </span>
      <div className="flex items-center gap-1">
        <button
          onClick={() => onChange(page - 1)}
          disabled={page <= 1}
          className="px-2 py-1 rounded border border-black/10 disabled:opacity-30 hover:bg-black/5"
        >
          Prev
        </button>
        {start > 1 && <span className="px-1 text-black/40">...</span>}
        {pages.map((p) => (
          <button
            key={p}
            onClick={() => onChange(p)}
            className={`px-2 py-1 rounded border ${
              p === page
                ? 'bg-black text-white border-black'
                : 'border-black/10 hover:bg-black/5'
            }`}
          >
            {p}
          </button>
        ))}
        {end < totalPages && <span className="px-1 text-black/40">...</span>}
        <button
          onClick={() => onChange(page + 1)}
          disabled={page >= totalPages}
          className="px-2 py-1 rounded border border-black/10 disabled:opacity-30 hover:bg-black/5"
        >
          Next
        </button>
      </div>
    </div>
  );
}
