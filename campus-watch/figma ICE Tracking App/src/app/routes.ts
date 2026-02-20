import { createBrowserRouter } from "react-router";
import { Layout } from "./components/Layout";
import { HomeScreen } from "./components/HomeScreen";
import { MapScreen } from "./components/MapScreen";
import { ReportScreen } from "./components/ReportScreen";
import { AlertsScreen } from "./components/AlertsScreen";

export const router = createBrowserRouter([
  {
    path: "/",
    Component: Layout,
    children: [
      { index: true, Component: HomeScreen },
      { path: "map", Component: MapScreen },
      { path: "report", Component: ReportScreen },
      { path: "alerts", Component: AlertsScreen },
    ],
  },
]);
