import { Checkbox } from '@/components/ui/checkbox';
import { useCallback, useId } from 'react';
import { useTranslation } from 'react-i18next';
import { GlobalModelTypeKey, GlobalModelTypes } from './constants';

function ModelTypeCheckbox({
  type,
  checked,
  onToggle,
}: {
  type: GlobalModelTypeKey;
  checked: boolean;
  onToggle: (type: GlobalModelTypeKey, checked: boolean) => void;
}) {
  const { t } = useTranslation();
  const id = useId();
  const handleChange = useCallback(
    (value: boolean | 'indeterminate') => onToggle(type, value === true),
    [onToggle, type],
  );
  return (
    <label htmlFor={id} className="flex items-center gap-1.5 text-sm">
      <Checkbox id={id} checked={checked} onCheckedChange={handleChange} />
      {t(`globalModel.type.${type}`)}
    </label>
  );
}

/** Picks the model types of a global model. */
export function ModelTypeCheckboxes({
  value,
  onChange,
}: {
  value: GlobalModelTypeKey[];
  onChange: (value: GlobalModelTypeKey[]) => void;
}) {
  const handleToggle = useCallback(
    (type: GlobalModelTypeKey, checked: boolean) =>
      onChange(
        checked
          ? [...value.filter((item) => item !== type), type]
          : value.filter((item) => item !== type),
      ),
    [onChange, value],
  );
  return (
    <div className="flex flex-wrap gap-x-4 gap-y-2">
      {GlobalModelTypes.map(({ storedKey }) => (
        <ModelTypeCheckbox
          key={storedKey}
          type={storedKey}
          checked={value.includes(storedKey)}
          onToggle={handleToggle}
        />
      ))}
    </div>
  );
}
