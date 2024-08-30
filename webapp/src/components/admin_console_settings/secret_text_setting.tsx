import React, {useEffect, useRef, useState} from 'react';

import BaseSetting from './base_setting';

type Props = {
    id: string;
    name: string;
    helpText: string;
    onChange: (value: string) => void;
    value: string;
    disabled?: boolean;
};

const SecretTextSetting = (props: Props) => {
    const [value, setValue] = useState('');
    const mounted = useRef(false);

    useEffect(() => {
        if (mounted.current) {
            setValue(props.value);
            return;
        }

        if (props.value) {
            setValue('*'.repeat(32));
        }

        mounted.current = true;
    }, [props.value]);

    const handleChange = (newValue: string) => {
        setValue(newValue);
        props.onChange(newValue);
    };

    return (
        <BaseSetting
            {...props}
        >
            <input
                id={props.id}
                className='form-control'
                type='text'
                value={value}
                onChange={(e) => handleChange(e.target.value)}
                disabled={props.disabled}
            />
        </BaseSetting>
    );
};

export default SecretTextSetting;
