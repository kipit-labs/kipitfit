import { useState } from "react";
import { useTranslation } from "react-i18next";
import { Button } from "@kipit-labs/kipitidy/button";
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from "@kipit-labs/kipitidy/dialog";
import { FloatingInput } from "@kipit-labs/kipitidy/floating-input";

type Props = {
  title: string;
  label: string;
  initial?: string;
  saving?: boolean;
  onSave: (name: string) => void;
};

/** Asks for a name, e.g. a folder's. Controlled, so a menu item can open it. */
export function NameDialog({ open, onOpenChange, ...props }: Props & { open: boolean; onOpenChange: (open: boolean) => void }) {
  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent>
        {/* The content unmounts when closed, so each opening starts from initial. */}
        <NameForm {...props} />
      </DialogContent>
    </Dialog>
  );
}

function NameForm({ title, label, initial = "", saving, onSave }: Props) {
  const { t } = useTranslation();
  const [name, setName] = useState(initial);
  return (
    <form
      className="contents"
      onSubmit={(e) => {
        e.preventDefault();
        onSave(name);
      }}
    >
      <DialogHeader>
        <DialogTitle>{title}</DialogTitle>
        <DialogDescription className="sr-only">{label}</DialogDescription>
      </DialogHeader>
      <FloatingInput label={label} required maxLength={100} autoComplete="off" autoFocus value={name} onChange={(e) => setName(e.target.value)} />
      <DialogFooter>
        <Button type="submit" loading={saving}>
          {t("common.save")}
        </Button>
      </DialogFooter>
    </form>
  );
}
