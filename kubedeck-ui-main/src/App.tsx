import { Route, Routes, Navigate } from "react-router";
import styles from "./App.module.css";
import { AuthPage } from "./pages/AuthPage/AuthPage";
import { InterfacePage } from "./pages/InterfacePage/InterfacePage";
// import { MetricsPage } from "./pages/MetricsPage/MetricsPage";
import { EnhancedMetricsPage } from "./pages/EnhancedMetricsPage/EnhancedMetricsPage";
import { SettingsPage } from "./pages/SettingsPage/SettingsPage";
import { MainLayout } from "./MainPayout";
import { Navigation } from "./coms/Navigation/Navigation";
// import { Comp404 } from "./pages/Comp404/Comp404.tsx";

function App() {
    return (
        <>
            <div className={styles.section}>
                <main className={styles.content}>
                    <Routes>
                        <Route
                            element={
                                <MainLayout NavigationComponent={Navigation} />
                            }
                        >
                            <Route
                                path="/interface"
                                element={<InterfacePage />}
                            />
                            <Route
                                path="/metrics"
                                element={<EnhancedMetricsPage />}
                            />
                            {/* <Route path="/enhanced-metrics" element={<EnhancedMetricsPage />} /> */}
                            <Route
                                path="/settings"
                                element={<SettingsPage />}
                            />
                        </Route>

                        <Route path="/auth" element={<AuthPage />} />

                        <Route
                            path="/"
                            element={<Navigate to="/metrics" replace />}
                        />
                        <Route
                            path="*"
                            element={<Navigate to="/metrics" replace />}
                        />
                    </Routes>
                </main>
            </div>
        </>
    );
}

export default App;
