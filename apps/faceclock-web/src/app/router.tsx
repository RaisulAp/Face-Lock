import { createBrowserRouter, RouterProvider } from "react-router-dom";
import { HealthPage } from "../pages/HealthPage";

// Fase 0 only has one route. The route table is centralized here on
// purpose (Fase 0 § 2.7: "struktur route sudah disiapkan untuk guard
// permission") so Fase 5 adds permission-guarded routes to this same
// object instead of inventing a second router.
const router = createBrowserRouter([
  {
    path: "/",
    element: <HealthPage />,
  },
]);

export function AppRouter() {
  return <RouterProvider router={router} />;
}
