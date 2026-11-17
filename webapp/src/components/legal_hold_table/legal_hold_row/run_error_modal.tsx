import {Button} from '@mattermost/compass-ui/components/button';
import {Modal, ModalOverlay} from '@mattermost/compass-ui/components/modal';

interface Props {
    onHide: () => void;
    show: boolean;
}

const RunErrorModal = ({show, onHide}: Props) => {
    return (
        <ModalOverlay
            open={show}
            onClose={onHide}
        >
            <Modal
                open={show}
                onClose={onHide}
                title='Error running legal hold'
                size='small'
                headerDivider={false}
                footerDivider={false}
                footer={(
                    <Button
                        type='button'
                        onClick={onHide}
                    >
                        {'Close'}
                    </Button>
                )}
            >
                {'There was an error running the legal hold. Please check the logs for more details.'}
            </Modal>
        </ModalOverlay>
    );
};

export default RunErrorModal;
