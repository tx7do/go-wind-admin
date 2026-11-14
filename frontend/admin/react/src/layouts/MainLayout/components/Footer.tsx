import { usePreferencesStore } from '@go-wind/admin-react';

export const AppFooter = () => {
  const { copyright } = usePreferencesStore((state) => state.preferences);

  if (!copyright.enable) return null;

  return (
    <div className="text-center text-[color:var(--gowind-secondary-text)] text-sm py-3">
      © {copyright.date} {copyright.companyName}
      {copyright.icp && (
        <a
          href={copyright.icpLink}
          target="_blank"
          rel="noopener noreferrer"
          className="ml-2 hover:opacity-80 transition"
        >
          {copyright.icp}
        </a>
      )}
    </div>
  );
};

export default AppFooter;
