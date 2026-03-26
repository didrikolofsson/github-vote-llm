import {
  Breadcrumb,
  BreadcrumbItem,
  BreadcrumbLink,
  BreadcrumbList,
  BreadcrumbPage,
  BreadcrumbSeparator,
} from "@/components/ui/breadcrumb";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { Separator } from "@/components/ui/separator";
import {
  Sidebar,
  SidebarContent,
  SidebarFooter,
  SidebarGroup,
  SidebarGroupContent,
  SidebarGroupLabel,
  SidebarHeader,
  SidebarInset,
  SidebarMenu,
  SidebarMenuButton,
  SidebarMenuItem,
  SidebarProvider,
  SidebarRail,
  SidebarSeparator,
  SidebarTrigger,
} from "@/components/ui/sidebar";
import { getMe, listMyOrganizations, listOrgRepositories } from "@/lib/api";
import { useQuery } from "@tanstack/react-query";
import { Building2, ChevronsUpDown } from "lucide-react";
import { Link, Outlet, useLocation } from "react-router-dom";
import { useAuth } from "../lib/auth";
import { cn } from "@/lib/utils";

/** Static path segment → display label (unknown segments get title-cased). */
const SEGMENT_LABELS: Record<string, string> = {
  dashboard: "Dashboard",
  repositories: "Repositories",
  settings: "Settings",
  roadmap: "Roadmap",
};

function labelForSegment(segment: string): string {
  if (SEGMENT_LABELS[segment]) return SEGMENT_LABELS[segment];
  return segment
    .split("-")
    .filter(Boolean)
    .map((w) => w.charAt(0).toUpperCase() + w.slice(1).toLowerCase())
    .join(" ");
}

function breadcrumbsFromPath(pathname: string, repoDisplayLabel: string) {
  const segments = pathname.split("/").filter(Boolean);
  return segments.map((segment, i) => {
    const isLast = i === segments.length - 1;
    const prefixPath = `/${segments.slice(0, i + 1).join("/")}`;
    const isRepoIdSegment =
      segments[0] === "repositories" && i === 1 && /^\d+$/.test(segment);
    const label = isRepoIdSegment ? repoDisplayLabel : labelForSegment(segment);
    return {
      label,
      href: isLast ? undefined : prefixPath,
      key: prefixPath,
    };
  });
}

export default function Layout() {
  const { logout, user } = useAuth();
  const { pathname } = useLocation();
  const isDashboard = pathname === "/dashboard";
  const isRepositorySection = pathname.startsWith("/repositories");
  const isSettings = pathname === "/settings";

  const repoRouteMatch = pathname.match(/^\/repositories\/(\d+)/);
  const activeRepoId = repoRouteMatch ? parseInt(repoRouteMatch[1], 10) : NaN;

  const { data: orgs = [] } = useQuery({
    queryKey: ["organizations"],
    queryFn: () => listMyOrganizations(),
  });
  const { data: profile } = useQuery({
    queryKey: ["users", "me"],
    queryFn: () => getMe(),
  });

  const org = orgs[0];
  const orgId = org?.id;
  const displayName = profile?.username ?? user?.email ?? "";

  const { data: repos = [] } = useQuery({
    queryKey: ["organizations", orgId, "repositories"],
    queryFn: () => listOrgRepositories(orgId!),
    enabled: !!orgId && Number.isFinite(activeRepoId),
  });

  const activeRepo = Number.isFinite(activeRepoId)
    ? repos.find((r) => r.id === activeRepoId)
    : undefined;
  const repoDisplayLabel = activeRepo
    ? `${activeRepo.owner}/${activeRepo.name}`
    : "Repository";

  const breadcrumb = breadcrumbsFromPath(pathname, repoDisplayLabel);

  return (
    <SidebarProvider>
      <Sidebar>
        <SidebarHeader>
          <SidebarMenu>
            <SidebarMenuItem>
              <SidebarMenuButton size="lg" asChild>
                <Link to="/dashboard">
                  <div className="flex aspect-square size-8 items-center justify-center rounded-md bg-sidebar-primary text-sidebar-primary-foreground">
                    <Building2 className="size-4" />
                  </div>
                  <div className="flex flex-col gap-0.5 leading-none">
                    <span className="font-semibold truncate">
                      {org?.name ?? "Organization"}
                    </span>
                    <span className="text-xs text-muted-foreground">Workspace</span>
                  </div>
                </Link>
              </SidebarMenuButton>
            </SidebarMenuItem>
          </SidebarMenu>
        </SidebarHeader>

        <SidebarContent>
          <SidebarGroup>
            <SidebarGroupLabel>Workspace</SidebarGroupLabel>
            <SidebarGroupContent>
              <SidebarMenu>
                <SidebarMenuItem>
                  <SidebarMenuButton asChild isActive={!!isDashboard}>
                    <Link to="/dashboard">
                      Dashboard
                    </Link>
                  </SidebarMenuButton>
                </SidebarMenuItem>
                <SidebarMenuItem>
                  <SidebarMenuButton asChild isActive={isRepositorySection}>
                    <Link to="/repositories">
                      Repositories
                    </Link>
                  </SidebarMenuButton>
                </SidebarMenuItem>
              </SidebarMenu>
            </SidebarGroupContent>
          </SidebarGroup>
        </SidebarContent>

        <SidebarSeparator />

        <SidebarFooter>
          <SidebarMenu>
            <SidebarMenuItem>
              <DropdownMenu>
                <DropdownMenuTrigger asChild>
                  <SidebarMenuButton
                    size="lg"
                    className="data-[state=open]:bg-sidebar-accent data-[state=open]:text-sidebar-accent-foreground"
                  >
                    <div className="flex aspect-square size-8 items-center justify-center rounded-full bg-muted text-sm font-semibold shrink-0">
                      {displayName.charAt(0).toUpperCase()}
                    </div>
                    <div className="flex flex-col gap-0.5 leading-none min-w-0">
                      <span className="text-xs truncate">
                        {displayName}
                      </span>
                    </div>
                    <ChevronsUpDown className="ml-auto shrink-0" />
                  </SidebarMenuButton>
                </DropdownMenuTrigger>
                <DropdownMenuContent
                  align="start"
                  side="top"
                  className="w-(--radix-dropdown-menu-trigger-width)"
                >
                  <DropdownMenuItem asChild className={
                    cn(isSettings ? "bg-sidebar-accent text-sidebar-accent-foreground font-medium" : "")
                  }>
                    <Link to="/settings">
                      Settings
                    </Link>
                  </DropdownMenuItem>
                  <DropdownMenuSeparator />
                  <DropdownMenuItem onClick={() => void logout()}>
                    Sign out
                  </DropdownMenuItem>
                </DropdownMenuContent>
              </DropdownMenu>
            </SidebarMenuItem>
          </SidebarMenu>
        </SidebarFooter>

        <SidebarRail />
      </Sidebar>

      <SidebarInset>
        <header className="flex h-12 shrink-0 items-center gap-2 border-b px-4">
          <SidebarTrigger className="-ml-1" />
          <Separator
            orientation="vertical"
            className="mr-2 data-vertical:h-4 data-vertical:self-auto"
          />
          <Breadcrumb>
            <BreadcrumbList>
              {breadcrumb.map((crumb, i) => (
                <BreadcrumbItem key={crumb.key}>
                  {i > 0 && <BreadcrumbSeparator />}
                  {crumb.href ? (
                    <BreadcrumbLink asChild>
                      <Link to={crumb.href}>{crumb.label}</Link>
                    </BreadcrumbLink>
                  ) : (
                    <BreadcrumbPage>{crumb.label}</BreadcrumbPage>
                  )}
                </BreadcrumbItem>
              ))}
            </BreadcrumbList>
          </Breadcrumb>
        </header>
        <div className="flex-1 p-8 max-w-[1280px] w-full mx-auto">
          <Outlet />
        </div>
      </SidebarInset>
    </SidebarProvider>
  );
}
