import { type AccountStatus, useAccount } from "@/lib/account";
import { cn } from "@/lib/utils";
import { useState } from "react";
import { useNavigate } from "react-router-dom";

const STATUS_OPTIONS: {
  value: AccountStatus;
  label: string;
  path: string;
}[] = [
  { value: "inactive", label: "Inactive", path: "/setup/connect-github" },
  {
    value: "github_connected",
    label: "GitHub connected",
    path: "/setup/install-app",
  },
  { value: "active", label: "Active", path: "/dashboard" },
  { value: "suspended", label: "Suspended", path: "/account/suspended" },
];

export function DevAccountWidget() {
  const { status, setStatus } = useAccount();
  const navigate = useNavigate();
  const [open, setOpen] = useState(false);

  function apply(option: (typeof STATUS_OPTIONS)[number]) {
    setStatus(option.value);
    navigate(option.path);
    setOpen(false);
  }

  return (
    <div className="fixed bottom-4 right-4 z-50 flex flex-col items-end gap-2">
      {open && (
        <div className="bg-popover border rounded-lg shadow-lg p-3 flex flex-col gap-1.5 min-w-44 animate-slide-up">
          <p className="text-[10px] font-semibold uppercase tracking-wider text-muted-foreground px-1 pb-0.5">
            Mock account status
          </p>
          {STATUS_OPTIONS.map((opt) => (
            <button
              key={opt.value}
              onClick={() => apply(opt)}
              className={cn(
                "text-left text-xs px-2.5 py-1.5 rounded-md transition-colors",
                status === opt.value
                  ? "bg-primary text-primary-foreground font-medium"
                  : "hover:bg-muted text-foreground",
              )}
            >
              {opt.label}
            </button>
          ))}
        </div>
      )}
      <button
        onClick={() => setOpen((o) => !o)}
        className={cn(
          "h-8 px-3 rounded-full text-[10px] font-bold uppercase tracking-wider border shadow-md transition-colors select-none",
          open
            ? "bg-primary text-primary-foreground border-primary"
            : "bg-background text-muted-foreground border-border hover:border-foreground/30 hover:text-foreground",
        )}
      >
        DEV
      </button>
    </div>
  );
}
