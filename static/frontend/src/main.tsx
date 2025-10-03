import { createRoot } from "react-dom/client";
import App from "./App.tsx";
import "./index.css";
import { SettingsProvider } from "@/context/settings.tsx";

createRoot(document.getElementById("beamdrop")!).render(
  <SettingsProvider>
    <App />
  </SettingsProvider>,
);
