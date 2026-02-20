import { RouterProvider } from "react-router";
import { Toaster } from "sonner";
import { router } from "./routes";

export default function App() {
  return (
    <div className="h-full w-full">
      <RouterProvider router={router} />
      <Toaster
        position="top-center"
        toastOptions={{
          style: {
            background: "#14141f",
            border: "1px solid rgba(255,255,255,0.08)",
            color: "#f0f0f5",
          },
        }}
      />
    </div>
  );
}
