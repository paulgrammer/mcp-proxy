import { useState } from "react"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select"
import { Card } from "@/components/ui/card"
import { Plus, Trash2, Key } from "lucide-react"
import type { Header } from "../types"
import { ConfirmationDialog } from "./confirmation-dialog"

interface HeadersSectionProps {
  headers: Header[]
  onUpdate: (headers: Header[]) => void
  onMarkChanged: () => void
}

export function HeadersSection({ headers, onUpdate, onMarkChanged }: HeadersSectionProps) {
  const [deleteIndex, setDeleteIndex] = useState<number | null>(null)

  const addHeader = () => {
    const newHeader: Header = {
      type: "constant",
      name: "",
      value: "",
    }
    onUpdate([...headers, newHeader])
    onMarkChanged()
  }

  const removeHeader = (index: number) => {
    onUpdate(headers.filter((_, i) => i !== index))
    setDeleteIndex(null)
    onMarkChanged()
  }

  const updateHeader = (index: number, field: keyof Header, value: string) => {
    const updated = [...headers]
    updated[index] = { ...updated[index], [field]: value }
    onUpdate(updated)
    onMarkChanged()
  }

  return (
    <>
      <div className="space-y-4">
        <div className="flex justify-between items-center">
          <h4 className="font-medium">Default Headers</h4>
          <Button onClick={addHeader} size="sm">
            <Plus className="h-4 w-4 mr-2" />
            Add Header
          </Button>
        </div>

        {headers.length === 0 ? (
          <div className="text-center py-8 text-muted-foreground border border-dashed rounded-lg">
            <Key className="h-12 w-12 mx-auto mb-3 opacity-50" />
            <p>No headers configured</p>
            <p className="text-sm">Add headers that will be sent with every request</p>
          </div>
        ) : (
          <div className="space-y-3">
            {headers.map((header, index) => (
              <Card key={index} className="p-4 transition-all duration-200 hover:shadow-sm">
                <div className="flex justify-between items-start mb-4">
                  <div className="flex items-center gap-2">
                    <Key className="h-4 w-4" />
                    <span className="font-medium text-sm">{header.name || `Header ${index + 1}`}</span>
                  </div>
                  <Button variant="ghost" size="sm" onClick={() => setDeleteIndex(index)}>
                    <Trash2 className="h-4 w-4" />
                  </Button>
                </div>

                <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
                  <div>
                    <Label>Type</Label>
                    <Select value={header.type} onValueChange={(value) => updateHeader(index, "type", value)}>
                      <SelectTrigger>
                        <SelectValue />
                      </SelectTrigger>
                      <SelectContent>
                        <SelectItem value="constant">Constant</SelectItem>
                        <SelectItem value="dynamic">Dynamic</SelectItem>
                      </SelectContent>
                    </Select>
                  </div>

                  <div>
                    <Label>Name *</Label>
                    <Input
                      value={header.name}
                      onChange={(e) => updateHeader(index, "name", e.target.value)}
                      placeholder="Content-Type"
                    />
                  </div>

                  <div>
                    <Label>Value *</Label>
                    <Input
                      value={header.value}
                      onChange={(e) => updateHeader(index, "value", e.target.value)}
                      placeholder="application/json"
                    />
                  </div>
                </div>
              </Card>
            ))}
          </div>
        )}
      </div>

      <ConfirmationDialog
        open={deleteIndex !== null}
        onOpenChange={(open) => !open && setDeleteIndex(null)}
        title="Delete Header"
        description={`Are you sure you want to delete the header "${
          deleteIndex !== null ? headers[deleteIndex]?.name || `Header ${deleteIndex + 1}` : ""
        }"? This action cannot be undone.`}
        onConfirm={() => deleteIndex !== null && removeHeader(deleteIndex)}
      />
    </>
  )
}
