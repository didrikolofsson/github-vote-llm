import { BadgeColor } from "@/components/ui/badge";
import { clsx, type ClassValue } from "clsx";
import { twMerge } from "tailwind-merge";
import { OrganizationMemberRole } from "./api-schemas";

export function cn(...inputs: ClassValue[]) {
  return twMerge(clsx(inputs));
}

export function slugify(s: string): string {
  return s
    .toLowerCase()
    .replace(/[^a-z0-9\s]/g, "")
    .trim()
    .replace(/\s+/g, "-");
}

export const userRoleColors: Record<OrganizationMemberRole, BadgeColor> = {
  owner: "indigo",
  member: "zinc",
} as const;
export function userRoleToBadgeColor(role: OrganizationMemberRole): BadgeColor {
  return userRoleColors[role];
}
