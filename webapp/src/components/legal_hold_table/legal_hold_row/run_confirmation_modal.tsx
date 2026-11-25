import ConfirmDialog from '@/components/confirm_dialog';

interface Props {
    onHide: () => void;
    onConfirm: () => void;
    show: boolean;
}

const RunConfirmationModal = ({show, onHide, onConfirm}: Props) => {
    return (
        <ConfirmDialog
            open={show}
            title='Confirm legal hold run'
            message='Are you sure you want to run this legal hold now?'
            confirmLabel='Run now'
            onConfirm={onConfirm}
            onCancel={onHide}
        />
    );
};

export default RunConfirmationModal;
