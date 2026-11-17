import type {ReactNode} from 'react';

import {Button} from '@mattermost/compass-ui/components/button';
import {Modal, ModalOverlay} from '@mattermost/compass-ui/components/modal';
import {SectionNotice} from '@mattermost/compass-ui/components/section-notice';

type Props = {
    open: boolean;
    title: string;
    message: ReactNode;
    confirmLabel: string;
    cancelLabel?: string;
    destructive?: boolean;
    loading?: boolean;
    error?: string;
    onConfirm: () => void;
    onCancel: () => void;
};

// Compass has no confirm dialog, so this composes one from Modal and two buttons.
const ConfirmDialog = (props: Props) => {
    return (
        <ModalOverlay
            open={props.open}
            onClose={props.onCancel}
        >
            <Modal
                open={props.open}
                onClose={props.onCancel}
                title={props.title}
                size='small'
                headerDivider={false}
                footerDivider={false}
                footer={(
                    <>
                        <Button
                            type='button'
                            emphasis='tertiary'
                            onClick={props.onCancel}
                        >
                            {props.cancelLabel ?? 'Cancel'}
                        </Button>
                        <Button
                            type='button'
                            destructive={props.destructive}
                            loading={props.loading}
                            onClick={props.onConfirm}
                        >
                            {props.confirmLabel}
                        </Button>
                    </>
                )}
            >
                {props.error && (
                    <SectionNotice
                        role='alert'
                        type='danger'
                        title={props.error}
                    />
                )}
                {props.message}
            </Modal>
        </ModalOverlay>
    );
};

export default ConfirmDialog;
