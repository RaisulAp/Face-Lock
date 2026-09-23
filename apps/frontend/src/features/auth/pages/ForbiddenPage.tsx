import { useLocation, Link } from "react-router-dom";
import { useTranslation } from "react-i18next";
import { ShieldAlert, ArrowLeft, Home } from "lucide-react";
import { Button } from "../../../components/ui/Button";

export function ForbiddenPage() {
  const { t } = useTranslation(["auth", "common"]);
  const location = useLocation();
  const requiredPermission = (location.state as { requiredPermission?: string })?.requiredPermission;

  return (
    <div className="min-h-screen flex items-center justify-center p-4 bg-slate-50">
      <div className="max-w-md w-full bg-white rounded-2xl border border-gray-200/80 p-8 shadow-xs text-center">
        <div className="w-12 h-12 rounded-2xl bg-amber-50 border border-amber-200 text-amber-600 flex items-center justify-center mx-auto mb-4">
          <ShieldAlert className="w-6 h-6" />
        </div>

        <h1 className="text-xl font-bold text-gray-900">{t("forbidden.title", { ns: "auth" })}</h1>
        <p className="text-xs text-gray-600 mt-2 leading-relaxed">
          {t("forbidden.message", { ns: "auth" })}
        </p>

        {requiredPermission && (
          <div className="mt-4 p-2.5 bg-gray-50 rounded-lg border border-gray-100 text-xs font-mono text-gray-700">
            {t("forbidden.requiredPermission", { ns: "auth" })}: <span className="font-semibold text-indigo-600">{requiredPermission}</span>
          </div>
        )}

        <div className="mt-6 flex items-center justify-center gap-3">
          <Button variant="outline" size="sm" onClick={() => window.history.back()}>
            <ArrowLeft className="w-4 h-4 mr-1" /> {t("actions.back", { ns: "common" })}
          </Button>
          <Link to="/">
            <Button variant="primary" size="sm">
              <Home className="w-4 h-4 mr-1" /> {t("forbidden.backHome", { ns: "auth" })}
            </Button>
          </Link>
        </div>
      </div>
    </div>
  );
}
