import { getHintText } from "../../lib/errors/hints";
import { Badge } from "../ui/Badge";

interface HintListProps {
  hints?: string[];
  className?: string;
}

export function HintList({ hints, className = "" }: HintListProps) {
  if (!hints || hints.length === 0) return null;

  return (
    <div className={`flex flex-wrap gap-1.5 ${className}`}>
      {hints.map((hint, idx) => (
        <Badge key={idx} variant="warning" size="sm">
          {getHintText(hint)}
        </Badge>
      ))}
    </div>
  );
}
