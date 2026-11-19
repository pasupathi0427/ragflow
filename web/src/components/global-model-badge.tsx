import i18n from '@/locales/config';

/** Marks a model the organization's admin provides to every user. */
export function GlobalModelBadge() {
  return (
    <span className="text-text-secondary bg-bg-card rounded-sm px-1 flex-shrink-0">
      {i18n.t('globalModel.badge')}
    </span>
  );
}
