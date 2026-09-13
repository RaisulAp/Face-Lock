import { useState } from "react";
import { Link, NavLink, Outlet, useLocation, useNavigate } from "react-router-dom";
import { useAuth } from "../lib/auth/useAuth";
import { NAVIGATION_CONFIG } from "./routes.config";
import { env } from "../lib/env";
import { Menu, X, LogOut, User, KeyRound, Shield, ChevronDown, Clock } from "lucide-react";
import { useQuery } from "@tanstack/react-query";
import { api } from "../lib/api";
import type { SuccessEnvelope } from "../types/api";

export function AppShell() {
    const { user, logout, can } = useAuth();
    const location = useLocation();
    const navigate = useNavigate();
    const [sidebarOpen, setSidebarOpen] = useState(false);
    const [userDropdownOpen, setUserDropdownOpen] = useState(false);

    // Poll pending count every 30s if user has attendance.approve or attendance.review
    const hasApprovePermission = can(["attendance.approve", "attendance.review"]);
    const { data: pendingData } = useQuery({
        queryKey: ["attendances", "pendingCount"],
        queryFn: () => api.getWithMeta<unknown[]>("/api/v1/attendances/pending?per_page=1"),
        enabled: hasApprovePermission,
        refetchInterval: 30_000,
    });

    const pendingCount = (pendingData as SuccessEnvelope<unknown[]>)?.meta?.total ?? 0;

    const handleLogout = async () => {
        await logout();
        navigate("/login");
    };

    const isDev = import.meta.env.DEV || env.apiBaseUrl.includes("localhost") || env.apiBaseUrl.includes("127.0.0.1");

    return (
        <div className="min-h-screen bg-slate-50 flex">
            {/* Mobile backdrop */}
            {sidebarOpen && (
                <div
                    className="fixed inset-0 z-40 bg-black/40 backdrop-blur-xs lg:hidden"
                    onClick={() => setSidebarOpen(false)}
                />
            )}

            {/* Sidebar */}
            <aside
                className={`fixed top-0 bottom-0 left-0 z-50 w-64 bg-slate-900 text-slate-300 flex flex-col transition-transform duration-200 ease-in-out lg:translate-x-0 ${sidebarOpen ? "translate-x-0" : "-translate-x-full"
                    }`}
            >
                {/* Brand */}
                <div className="h-16 flex items-center justify-between px-6 border-b border-slate-800">
                    <Link to="/" className="flex items-center gap-2.5">
                        <div className="w-8 h-8 rounded-lg bg-indigo-600 flex items-center justify-center font-black text-white text-base tracking-wider shadow-sm">
                            FC
                        </div>
                        <div>
                            <span className="font-bold text-white text-sm tracking-tight block">FaceClock</span>
                            <span className="text-[10px] text-slate-400 font-medium tracking-wide uppercase block -mt-1">
                                Admin Panel
                            </span>
                        </div>
                    </Link>

                    <button
                        type="button"
                        onClick={() => setSidebarOpen(false)}
                        className="lg:hidden text-slate-400 hover:text-white p-1 rounded-md"
                    >
                        <X className="w-5 h-5" />
                    </button>
                </div>

                {/* Navigation list */}
                <nav className="flex-1 overflow-y-auto px-4 py-4 space-y-6">
                    {NAVIGATION_CONFIG.map((group) => {
                        const visibleItems = group.items.filter((item) => can(item.permission));
                        if (visibleItems.length === 0) return null;

                        return (
                            <div key={group.title} className="space-y-1">
                                <p className="px-3 text-[11px] font-semibold uppercase tracking-wider text-slate-300">
                                    {group.title}
                                </p>
                                {visibleItems.map((item) => {
                                    const Icon = item.icon;
                                    const isActive =
                                        item.path === "/attendances"
                                            ? location.pathname === "/attendances" ||
                                            (location.pathname.startsWith("/attendances/") &&
                                                !location.pathname.startsWith("/attendances/pending"))
                                            : location.pathname === item.path ||
                                            (item.path !== "/" &&
                                                item.path !== "/dashboard" &&
                                                location.pathname.startsWith(item.path + "/"));

                                    const showBadge = item.badgeKey === "pendingCount" && pendingCount > 0;

                                    return (
                                        <NavLink
                                            key={item.path}
                                            to={item.path}
                                            onClick={() => setSidebarOpen(false)}
                                            className={`flex items-center justify-between px-3 py-2 rounded-lg text-xs font-medium transition-colors ${isActive
                                                ? "bg-indigo-600 text-white font-semibold shadow-xs"
                                                : "text-slate-300 hover:bg-slate-800 hover:text-white"
                                                }`}
                                        >
                                            <div className="flex items-center gap-2.5">
                                                <Icon className={`w-4 h-4 ${isActive ? "text-white" : "text-slate-400"}`} />
                                                <span>{item.label}</span>
                                            </div>
                                            {showBadge && (
                                                <span className="px-1.5 py-0.5 text-[10px] font-bold bg-amber-500 text-slate-950 rounded-full">
                                                    {pendingCount}
                                                </span>
                                            )}
                                        </NavLink>
                                    );
                                })}
                            </div>
                        );
                    })}
                </nav>

                {/* User Card at bottom of Sidebar */}
                <div className="p-4 border-t border-slate-800 bg-slate-950/40">
                    <div className="flex items-center gap-3">
                        <div className="w-8 h-8 rounded-full bg-slate-800 flex items-center justify-center text-slate-300 shrink-0">
                            <User className="w-4 h-4" />
                        </div>
                        <div className="flex-1 min-w-0">
                            <p className="text-xs font-semibold text-white truncate">{user?.full_name || user?.email}</p>
                            <div className="flex items-center gap-1 flex-wrap mt-0.5">
                                {user?.roles?.map((r: any, idx: number) => {
                                    const roleName = typeof r === "string" ? r : r?.name || "";
                                    const roleKey = typeof r === "string" ? r : r?.id || `${roleName}-${idx}`;
                                    return (
                                        <span
                                            key={roleKey}
                                            className="text-[9px] font-medium bg-slate-800 text-slate-300 px-1.5 py-0.2 rounded"
                                        >
                                            {roleName}
                                        </span>
                                    );
                                })}
                            </div>
                        </div>
                    </div>
                </div>
            </aside>

            {/* Main Content Area */}
            <div className="flex-1 flex flex-col min-w-0 lg:pl-64">
                {/* Topbar */}
                <header className="h-16 bg-white border-b border-gray-200/80 px-4 sm:px-8 flex items-center justify-between sticky top-0 z-30 shadow-2xs">
                    <div className="flex items-center gap-3">
                        <button
                            type="button"
                            onClick={() => setSidebarOpen(true)}
                            className="lg:hidden p-2 rounded-lg text-gray-600 hover:bg-gray-100 cursor-pointer"
                        >
                            <Menu className="w-5 h-5" />
                        </button>
                        <span className="text-xs font-semibold text-gray-500 hidden sm:inline">FaceClock System</span>
                    </div>

                    <div className="flex items-center gap-4">
                        {/* Environment Badge */}
                        <span
                            className={`text-[10px] uppercase font-bold tracking-wider px-2 py-0.5 rounded-full border ${isDev
                                ? "bg-amber-50 text-amber-700 border-amber-300"
                                : "bg-emerald-50 text-emerald-700 border-emerald-300"
                                }`}
                        >
                            {isDev ? "Development" : "Production"}
                        </span>

                        {/* Profile Menu */}
                        <div className="relative">
                            <button
                                type="button"
                                onClick={() => setUserDropdownOpen(!userDropdownOpen)}
                                className="flex items-center gap-2 text-xs font-medium text-gray-700 hover:text-gray-900 p-1.5 rounded-lg hover:bg-gray-100 cursor-pointer"
                            >
                                <div className="w-7 h-7 rounded-full bg-indigo-100 text-indigo-700 flex items-center justify-center font-bold">
                                    {(user?.email?.[0] || "U").toUpperCase()}
                                </div>
                                <span className="hidden md:inline">{user?.email}</span>
                                <ChevronDown className="w-3.5 h-3.5 text-gray-400" />
                            </button>

                            {userDropdownOpen && (
                                <>
                                    <div className="fixed inset-0 z-40" onClick={() => setUserDropdownOpen(false)} />
                                    <div className="absolute right-0 mt-2 w-56 bg-white rounded-xl shadow-lg border border-gray-100 py-1.5 z-50 text-xs text-gray-700">
                                        <div className="px-3.5 py-2 border-b border-gray-100">
                                            <p className="font-semibold text-gray-900 truncate">
                                                {user?.full_name || "Administrator"}
                                            </p>
                                            <p className="text-[11px] text-gray-500 truncate">{user?.email}</p>
                                        </div>

                                        <Link
                                            to="/account"
                                            onClick={() => setUserDropdownOpen(false)}
                                            className="flex items-center gap-2 px-3.5 py-2 hover:bg-gray-50 text-gray-700"
                                        >
                                            <KeyRound className="w-3.5 h-3.5 text-gray-400" />
                                            <span>Ganti Password & Sesi</span>
                                        </Link>

                                        <Link
                                            to="/portal/attendance"
                                            onClick={() => setUserDropdownOpen(false)}
                                            className="flex items-center gap-2 px-3.5 py-2 hover:bg-indigo-50 text-indigo-700 font-medium"
                                        >
                                            <Clock className="w-3.5 h-3.5 text-indigo-600" />
                                            <span>Portal Absensi Karyawan</span>
                                        </Link>

                                        {can("audit.read") && (
                                            <Link
                                                to="/audit-logs"
                                                onClick={() => setUserDropdownOpen(false)}
                                                className="flex items-center gap-2 px-3.5 py-2 hover:bg-gray-50 text-gray-700"
                                            >
                                                <Shield className="w-3.5 h-3.5 text-gray-400" />
                                                <span>Audit Log Aktivitas</span>
                                            </Link>
                                        )}

                                        <div className="border-t border-gray-100 my-1" />

                                        <button
                                            type="button"
                                            onClick={() => {
                                                setUserDropdownOpen(false);
                                                handleLogout();
                                            }}
                                            className="w-full flex items-center gap-2 px-3.5 py-2 text-rose-600 hover:bg-rose-50 text-left font-medium cursor-pointer"
                                        >
                                            <LogOut className="w-3.5 h-3.5" />
                                            <span>Keluar (Logout)</span>
                                        </button>
                                    </div>
                                </>
                            )}
                        </div>
                    </div>
                </header>

                {/* Page body */}
                <main className="flex-1 p-4 sm:p-8 max-w-7xl w-full mx-auto">
                    <Outlet />
                </main>
            </div>
        </div>
    );
}
