import React from "react";
import { Dialog } from "./Dialog";

export interface ModalProps {
    open: boolean;
    onClose: () => void;
    title: string;
    description?: string;
    children: React.ReactNode;
    maxWidth?: "sm" | "md" | "lg" | "xl" | "2xl";
}

export function Modal({
    open,
    onClose,
    title,
    description,
    children,
    maxWidth = "md",
}: ModalProps) {
    return (
        <Dialog
            isOpen={open}
            onClose={onClose}
            title={title}
            description={description}
            maxWidth={maxWidth}
        >
            {children}
        </Dialog>
    );
}
