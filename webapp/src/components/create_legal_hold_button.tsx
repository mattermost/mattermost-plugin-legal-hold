import React from 'react';

interface CreateLegalHoldButtonProps {
    onClick: () => void;
    dataTestId?: string;
}

const CreateLegalHoldButton = (props: CreateLegalHoldButtonProps) => {
    return (
        <button
            type='submit'
            data-testid={props.dataTestId}
            id='createLegalHold'
            className='btn btn-primary'
            onClick={props.onClick}
        >
            {'Create new'}
        </button>
    );
};

export default CreateLegalHoldButton;
