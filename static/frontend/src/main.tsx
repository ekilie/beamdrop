import { createRoot } from "react-dom/client";
import App from "./App.tsx";
import "./index.css";
import { SettingsProvider } from "@/context/settings.tsx";
import { installCSRFInterceptor } from "@/lib/csrf";

// Install CSRF token interceptor before any fetch calls
installCSRFInterceptor();

createRoot(document.getElementById("beamdrop")!).render(
  <SettingsProvider>
    <App />
  </SettingsProvider>,
);
