import axios from "axios";

export const API_URL = __API_URL__;

const instance = axios.create({
	baseURL: API_URL,
	timeout: 1000 * 5
});

export default instance;
