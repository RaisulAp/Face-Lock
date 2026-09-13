import { Link, NavLink, Outlet, useLocation, useNavigate } from "react-router-dom";
import { useAuth } from "../lib/auth/useAuth";
import { Camera, UserCheck, History, FileCheck2, User, LogOut, LayoutDashboard } from "lucide-react";

export function EmployeeShell() {
    const { user, logout, can } = useAuth();
    const location = useLocation();
    const navigate = useNavigate();

    const handleLogout = async () => {
        await logout();
        navigate("/login");
    };

    const canAccessAdmin =
        can("attendance.read_all") ||
        can("attendance.approve") ||
        can("employee.read") ||
        can("settings.read") ||
        can("role.read");

    const navItems = [
        {
            to: "/portal/attendance",
            label: "Presensi",
            icon: Camera,
        },
        {
            to: "/portal/enrollment",
            label: "Wajah",
            icon: UserCheck,
        },
        {
            to: "/portal/history",
            label: "Riwayat",
            icon: History,
        },
        {
            to: "/portal/consent",
            label: "Persetujuan",
            icon: FileCheck2,
        },
        {
            to: "/portal/profile",
            label: "Profil",
            icon: User,
        },
    ];

    return (
        <div className="min-h-screen bg-slate-50 dark:bg-slate-950 flex flex-col text-slate-900 dark:text-slate-100">
            {/* Top Header */}
            <header className="sticky top-0 z-40 bg-white/90 dark:bg-slate-900/90 backdrop-blur-md border-b border-slate-200 dark:border-slate-800 transition-colors">
                <div className="max-w-4xl mx-auto px-4 h-16 flex items-center justify-between">
                    <Link to="/portal/attendance" className="flex items-center gap-2.5">
                        <div className="w-8 h-8 rounded-xl bg-indigo-600 flex items-center justify-center font-black text-white text-sm shadow-md shadow-indigo-500/20">
                            FC
                        </div>
                        <div>
                            <span className="font-bold text-slate-900 dark:text-white text-sm tracking-tight block">
                                FaceClock
                            </span>
                            <span className="text-[10px] text-slate-500 dark:text-slate-400 font-medium tracking-wide uppercase block -mt-1">
                                Portal Karyawan
                            </span>
                        </div>
                    </Link>

                    {/* Desktop Navigation Links */}
                    <nav className="hidden md:flex items-center gap-1">
                        {navItems.map((item) => {
                            const Icon = item.icon;
                            const isActive = location.pathname.startsWith(item.to);
                            return (
                                <NavLink
                                    key={item.to}
                                    to={item.to}
                                    className={`flex items-center gap-1.5 px-3 py-1.5 rounded-lg text-xs font-semibold transition-all ${isActive
                                            ? "bg-indigo-50 dark:bg-indigo-950/50 text-indigo-600 dark:text-indigo-400 font-bold"
                                            : "text-slate-600 dark:text-slate-400 hover:text-slate-900 dark:hover:text-white hover:bg-slate-100 dark:hover:bg-slate-800"
                                        }`}
                                >
                                    <Icon className="w-3.5 h-3.5" />
                                    <span>{item.label}</span>
                                </NavLink>
                            );
                        })}
                    </nav>

                    {/* Right Action Icons */}
                    <div className="flex items-center gap-2">
                        {user && (
                            <span className="hidden sm:inline-block text-xs font-medium text-slate-600 dark:text-slate-300 mr-1 max-w-[140px] truncate">
                                {user.full_name || user.email}
                            </span>
                        )}

                        {canAccessAdmin && (
                            <Link
                                to="/dashboard"
                                className="inline-flex items-center gap-1 px-2.5 py-1.5 rounded-lg text-xs font-medium text-slate-600 dark:text-slate-300 hover:text-indigo-600 dark:hover:text-indigo-400 hover:bg-slate-100 dark:hover:bg-slate-800 border border-slate-200 dark:border-slate-700 transition-colors"
                                title="Beralih ke Panel Admin"
                            >
                                <LayoutDashboard className="w-3.5 h-3.5" />
                                <span className="hidden sm:inline">Panel Admin</span>
                            </Link>
                        )}

                        <button
                            type="button"
                            onClick={handleLogout}
                            className="p-2 rounded-lg text-slate-500 hover:text-rose-600 dark:text-slate-400 dark:hover:text-rose-400 hover:bg-slate-100 dark:hover:bg-slate-800 transition-colors"
                            title="Keluar / Logout"
                        >
                            <LogOut className="w-4 h-4" />
                        </button>
                    </div>
                </div>
            </header>

            {/* Main Content Area */}
            <main className="flex-1 max-w-4xl w-full mx-auto p-4 pb-24 md:pb-8">
                <Outlet />
            </main>

            {/* Mobile Bottom Navigation Bar */}
            <nav className="md:hidden fixed bottom-0 left-0 right-0 z-40 bg-white/95 dark:bg-slate-900/95 backdrop-blur-md border-t border-slate-200 dark:border-slate-800 px-2 py-1 shadow-lg">
                <div className="grid grid-cols-5 gap-1">
                    {navItems.map((item) => {
                        const Icon = item.icon;
                        const isActive = location.pathname.startsWith(item.to);
                        return (
                            <NavLink
                                key={item.to}
                                to={item.to}
                                className={`flex flex-col items-center justify-center py-1.5 px-1 rounded-xl text-[10px] font-medium transition-all ${isActive
                                        ? "text-indigo-600 dark:text-indigo-400 font-bold"
                                        : "text-slate-500 dark:text-slate-400 hover:text-slate-900 dark:hover:text-white"
                                    }`}
                            >
                                <div
                                    className={`p-1 rounded-lg mb-0.5 transition-colors ${isActive ? "bg-indigo-50 dark:bg-indigo-950/60" : "bg-transparent"
                                        }`}
                                >
                                    <Icon className="w-5 h-5" />
                                </div>
                                <span className="truncate">{item.label}</span>
                            </NavLink>
                        );
                    })}
                </div>
            </nav>
        </div>
    );
}
