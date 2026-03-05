import { useEffect, useRef } from 'react'

interface ConfirmDialogProps {
  open: boolean
  title: string
  message: string
  confirmLabel?: string
  onConfirm: () => void
  onCancel: () => void
}

export default function ConfirmDialog({ open, title, message, confirmLabel = 'Delete', onConfirm, onCancel }: ConfirmDialogProps) {
  const dialogRef = useRef<HTMLDialogElement>(null)

  useEffect(() => {
    const dialog = dialogRef.current
    if (!dialog) return
    if (open && !dialog.open) {
      dialog.showModal()
    } else if (!open && dialog.open) {
      dialog.close()
    }
  }, [open])

  if (!open) return null

  return (
    <dialog
      ref={dialogRef}
      onClose={onCancel}
      className="bg-gray-900 border border-gray-700 rounded-lg p-0 backdrop:bg-black/60 text-white max-w-sm w-full overflow-hidden"
    >
      <div className="p-6 space-y-4">
        <h3 className="text-lg font-semibold">{title}</h3>
        <p className="text-sm text-gray-400 break-words">{message}</p>
        <div className="flex justify-end gap-3 pt-2">
          <button
            onClick={onCancel}
            className="px-4 py-2 text-sm bg-gray-800 text-gray-300 rounded hover:bg-gray-700"
          >
            Cancel
          </button>
          <button
            onClick={onConfirm}
            className="px-4 py-2 text-sm bg-red-600 text-white rounded hover:bg-red-700"
          >
            {confirmLabel}
          </button>
        </div>
      </div>
    </dialog>
  )
}
