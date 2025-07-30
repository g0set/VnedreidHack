import { Outlet } from "react-router";
import { type ComponentType, type FC } from "react";

interface MainLayoutProps {
  NavigationComponent: ComponentType;
}

export const MainLayout: FC<MainLayoutProps> = ({ NavigationComponent }) => {
  return (
    <div className="section">
      <NavigationComponent />
      <div className="section-info">
        <Outlet />
      </div>
    </div>
  );
};
