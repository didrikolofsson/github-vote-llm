import { BadgeColor } from "@/components/ui/badge";
import { clsx, type ClassValue } from "clsx";
import { twMerge } from "tailwind-merge";
import { OrganizationMemberRole } from "./api-schemas";

export function cn(...inputs: ClassValue[]) {
  return twMerge(clsx(inputs));
}

export const userRoleColors: Record<OrganizationMemberRole, BadgeColor> = {
  owner: "lime",
  member: "zinc",
} as const;
export function userRoleToBadgeColor(role: OrganizationMemberRole): BadgeColor {
  return userRoleColors[role];
}
