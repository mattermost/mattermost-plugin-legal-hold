import React from 'react';
import {Modal} from 'react-bootstrap';

interface Props {
    onHide: () => void;
    show: boolean;
}

const RunErrorModal = ({show, onHide}: Props) => {
    return (
        <Modal
            show={show}
            onHide={onHide}
            centered={true}
        >
            <Modal.Header closeButton={true}>
                <Modal.Title>{'Error Running Legal Hold'}</Modal.Title>
            </Modal.Header>
            <Modal.Body>
                {'There was an error running the legal hold. Please contact your system administrator.'}
            </Modal.Body>
            <Modal.Footer>
                <button
                    type='button'
                    className='btn btn-primary'
                    onClick={onHide}
                >
                    {'Close'}
                </button>
            </Modal.Footer>
        </Modal>
    );
};

export default RunErrorModal;
