import * as React from "react"
import { cn } from "@/lib/utils"

const colors = {
  red:     "bg-red-500/15 text-red-700 dark:bg-red-500/10 dark:text-red-400",
  orange:  "bg-orange-500/15 text-orange-700 dark:bg-orange-500/10 dark:text-orange-400",
  amber:   "bg-amber-400/20 text-amber-700 dark:bg-amber-400/10 dark:text-amber-400",
  yellow:  "bg-yellow-400/20 text-yellow-700 dark:bg-yellow-400/10 dark:text-yellow-300",
  lime:    "bg-lime-400/20 text-lime-700 dark:bg-lime-400/10 dark:text-lime-300",
  green:   "bg-green-500/15 text-green-700 dark:bg-green-500/10 dark:text-green-400",
  emerald: "bg-emerald-500/15 text-emerald-700 dark:bg-emerald-500/10 dark:text-emerald-400",
  teal:    "bg-teal-500/15 text-teal-700 dark:bg-teal-500/10 dark:text-teal-400",
  cyan:    "bg-cyan-400/20 text-cyan-700 dark:bg-cyan-400/10 dark:text-cyan-400",
  sky:     "bg-sky-500/15 text-sky-700 dark:bg-sky-500/10 dark:text-sky-400",
  blue:    "bg-blue-500/15 text-blue-700 dark:bg-blue-500/10 dark:text-blue-400",
  indigo:  "bg-indigo-500/15 text-indigo-700 dark:bg-indigo-500/10 dark:text-indigo-400",
  violet:  "bg-violet-500/15 text-violet-700 dark:bg-violet-500/10 dark:text-violet-400",
  purple:  "bg-purple-500/15 text-purple-700 dark:bg-purple-500/10 dark:text-purple-400",
  fuchsia: "bg-fuchsia-400/20 text-fuchsia-700 dark:bg-fuchsia-400/10 dark:text-fuchsia-400",
  pink:    "bg-pink-400/20 text-pink-700 dark:bg-pink-400/10 dark:text-pink-400",
  rose:    "bg-rose-400/15 text-rose-700 dark:bg-rose-400/10 dark:text-rose-400",
  zinc:    "bg-zinc-600/10 text-zinc-700 dark:bg-zinc-400/10 dark:text-zinc-400",
} as const

export type BadgeColor = keyof typeof colors

function Badge({
  color = "zinc",
  small = false,
  className,
  ...props
}: React.ComponentProps<"span"> & { color?: BadgeColor, small?: boolean }) {
  return (
    <span
      data-slot="badge"
      className={cn(
        "inline-flex items-center gap-x-1.5 rounded-md px-1.5 py-0.5 text-sm/5 font-medium sm:text-xs/5 forced-colors:outline w-fit",
        small ? "text-xs/5" : "text-sm/5",
        small ? "px-1.5 py-0" : "px-2.5 py-0.5",
        colors[color],
        className
      )}
      {...props}
    />
  )
}

export { Badge }
