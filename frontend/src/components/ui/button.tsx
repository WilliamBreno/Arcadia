import { Button as ButtonPrimitive } from "@base-ui/react/button"
import { cva, type VariantProps } from "class-variance-authority"
import { cn } from "cn"

const buttonVariants = cva(
  "evve-button group/button inline-flex shrink-0 items-center justify-center rounded-xl border border-transparent bg-clip-padding text-sm font-semibold leading-snug text-center whitespace-normal transition-colors outline-none select-none focus-visible:ring-3 focus-visible:ring-ring/35 focus-visible:ring-offset-2 focus-visible:ring-offset-background disabled:pointer-events-none disabled:opacity-45 aria-invalid:border-destructive aria-invalid:ring-3 aria-invalid:ring-destructive/20",
  {
    variants: {
      variant: {
        default: "border-primary bg-primary text-primary-foreground shadow-sm hover:bg-primary/90 active:bg-primary/85",
        outline:
          "border-input bg-card text-foreground shadow-xs hover:border-primary/40 hover:bg-accent aria-expanded:border-primary/40 aria-expanded:bg-accent aria-expanded:text-accent-foreground",
        secondary:
          "bg-secondary text-secondary-foreground hover:bg-accent aria-expanded:bg-accent",
        ghost:
          "text-muted-foreground hover:bg-accent hover:text-accent-foreground aria-expanded:bg-accent aria-expanded:text-accent-foreground",
        destructive:
          "border-destructive/20 bg-destructive/5 text-destructive hover:border-destructive/40 hover:bg-destructive/10 focus-visible:ring-destructive/25 dark:bg-destructive/10",
        link: "text-primary underline-offset-4 hover:underline",
      },
      size: {
        default:
          "min-h-11 gap-2 px-4 py-2.5",
        xs: "min-h-7 gap-1 rounded-lg px-2 py-1 text-xs",
        sm: "min-h-9 gap-1.5 rounded-lg px-3 py-2 text-xs",
        lg: "min-h-12 gap-2 rounded-xl px-6 py-3 text-base",
        icon: "size-11 p-0",
        "icon-xs": "size-7 rounded-lg p-0",
        "icon-sm": "size-9 rounded-lg p-0",
        "icon-lg": "size-12 p-0",
      },
    },
    defaultVariants: {
      variant: "default",
      size: "default",
    },
  }
)

function Button({
  className,
  variant = "default",
  size = "default",
  ...props
}: ButtonPrimitive.Props & VariantProps<typeof buttonVariants>) {
  return (
    <ButtonPrimitive
      data-slot="button"
      data-variant={variant}
      data-size={size}
      className={cn(buttonVariants({ variant, size, className }))}
      {...props}
    />
  )
}

export { Button, buttonVariants }
